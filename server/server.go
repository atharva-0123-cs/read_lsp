package server

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/atharva-0123-cs/read_lsp/analysis"
	"github.com/atharva-0123-cs/read_lsp/lsp"
	"github.com/atharva-0123-cs/read_lsp/rpc"
)

const (
	defaultWorkers             = 4
	defaultDiagnosticsDebounce = 75 * time.Millisecond
	initialScannerBuffer       = 64 * 1024
	maxMessageSize             = 32 * 1024 * 1024
)

type Options struct {
	Logger              *log.Logger
	State               *analysis.State
	Workers             int
	DiagnosticsDebounce time.Duration
}

type job struct {
	base     rpc.BaseMessage
	contents []byte
}

type Server struct {
	logger           *log.Logger
	state            *analysis.State
	workers          int
	diagnosticsDelay time.Duration

	ctx    context.Context
	cancel context.CancelFunc

	out       chan any
	outMu     sync.Mutex
	outClosed bool
	writerWG  sync.WaitGroup

	jobs     chan job
	workerWG sync.WaitGroup

	cancelMu         sync.Mutex
	cancels          map[string]context.CancelFunc
	canceledRequests map[string]bool

	publishMu         sync.Mutex
	diagnosticsMu     sync.Mutex
	diagnosticsSeq    uint64
	diagnosticsLatest map[string]uint64
	diagnosticsTimers map[string]*time.Timer
}

func New(options Options) *Server {
	if options.Logger == nil {
		options.Logger = log.New(io.Discard, "", 0)
	}
	if options.State == nil {
		options.State = analysis.NewState()
	}
	if options.Workers <= 0 {
		options.Workers = defaultWorkers
	}
	if options.DiagnosticsDebounce < 0 {
		options.DiagnosticsDebounce = 0
	}

	return &Server{
		logger:            options.Logger,
		state:             options.State,
		workers:           options.Workers,
		diagnosticsDelay:  options.DiagnosticsDebounce,
		cancels:           map[string]context.CancelFunc{},
		canceledRequests:  map[string]bool{},
		diagnosticsLatest: map[string]uint64{},
		diagnosticsTimers: map[string]*time.Timer{},
	}
}

func (s *Server) Serve(ctx context.Context, reader io.Reader, writer io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.out = make(chan any, 128)
	s.jobs = make(chan job, s.workers*8)

	s.startWriter(writer)
	s.startWorkers()

	scanner := bufio.NewScanner(reader)
	scanner.Split(rpc.Split)
	scanner.Buffer(make([]byte, initialScannerBuffer), maxMessageSize)

	shouldExit := false
	for scanner.Scan() {
		message := append([]byte(nil), scanner.Bytes()...)
		base, contents, err := rpc.DecodeMessageBase(message)
		if err != nil {
			s.logger.Printf("could not decode message: %s", err)
			continue
		}

		if base.Method == "exit" {
			shouldExit = true
			break
		}

		if base.IsRequest() {
			if isAsyncRequest(base.Method) {
				s.dispatch(base, contents)
			} else {
				s.handleRequest(context.Background(), job{base: base, contents: contents})
			}
			continue
		}

		s.handleNotification(base.Method, contents)
	}

	err := scanner.Err()
	s.shutdown(shouldExit)
	return err
}

func (s *Server) startWriter(writer io.Writer) {
	s.writerWG.Add(1)
	go func() {
		defer s.writerWG.Done()
		for message := range s.out {
			if _, err := io.WriteString(writer, rpc.EncodeMessage(message)); err != nil {
				s.logger.Printf("could not write response: %s", err)
			}
		}
	}()
}

func (s *Server) startWorkers() {
	for idx := 0; idx < s.workers; idx++ {
		s.workerWG.Add(1)
		go s.worker()
	}
}

func (s *Server) worker() {
	defer s.workerWG.Done()

	for current := range s.jobs {
		requestCtx, cancel := context.WithCancel(s.ctx)
		key := current.base.RequestKey()
		if key != "" {
			s.registerCancel(key, cancel)
		}

		s.handleRequest(requestCtx, current)

		if key != "" {
			s.unregisterCancel(key)
		}
		cancel()
	}
}

func isAsyncRequest(method string) bool {
	switch method {
	case "textDocument/hover", "textDocument/definition", "textDocument/references",
		"textDocument/documentSymbol", "textDocument/foldingRange",
		"textDocument/codeAction", "textDocument/completion":
		return true
	default:
		return false
	}
}

func (s *Server) dispatch(base rpc.BaseMessage, contents []byte) {
	current := job{
		base:     base,
		contents: append([]byte(nil), contents...),
	}

	select {
	case s.jobs <- current:
	case <-s.ctx.Done():
	}
}

func (s *Server) handleNotification(method string, contents []byte) {
	switch method {
	case "initialized":
		s.logger.Print("client initialized")

	case "textDocument/didOpen":
		var notification lsp.DidOpenTextDocumentNotifiction
		if err := json.Unmarshal(contents, &notification); err != nil {
			s.logger.Printf("textDocument/didOpen: %s", err)
			return
		}

		textDocument := notification.Params.TextDocument
		s.state.PutDocument(textDocument.URI, textDocument.Text)
		s.scheduleDiagnostics(textDocument.URI)

	case "textDocument/didChange":
		var notification lsp.TextDocumentDidChangeNotification
		if err := json.Unmarshal(contents, &notification); err != nil {
			s.logger.Printf("textDocument/didChange: %s", err)
			return
		}

		uri := notification.Params.TextDocument.URI
		for _, change := range notification.Params.ContentChanges {
			s.state.ReplaceDocument(uri, change.Text)
		}
		s.scheduleDiagnostics(uri)

	case "textDocument/didClose":
		var notification lsp.DidCloseTextDocumentNotification
		if err := json.Unmarshal(contents, &notification); err != nil {
			s.logger.Printf("textDocument/didClose: %s", err)
			return
		}

		uri := notification.Params.TextDocument.URI
		s.state.CloseDocument(uri)
		s.clearDiagnostics(uri)

	case "$/cancelRequest":
		var notification lsp.CancelRequestNotification
		if err := json.Unmarshal(contents, &notification); err != nil {
			s.logger.Printf("$/cancelRequest: %s", err)
			return
		}
		s.cancelRequest(string(notification.Params.ID))

	default:
		s.logger.Printf("unsupported notification: %s", method)
	}
}

func (s *Server) handleRequest(ctx context.Context, current job) {
	base := current.base

	if s.requestCanceled(ctx, base.ID) {
		return
	}

	switch base.Method {
	case "initialize":
		var request lsp.InitializeRequest
		if err := json.Unmarshal(current.contents, &request); err != nil {
			s.sendError(base.ID, -32700, "parse error")
			return
		}

		if request.Params.ClientInfo != nil {
			s.logger.Printf(
				"connected to %s %s",
				request.Params.ClientInfo.Name,
				request.Params.ClientInfo.Version,
			)
		}

		s.enqueue(lsp.NewInitializeResponseWithID(base.ID))

	case "shutdown":
		var request lsp.Request
		if err := json.Unmarshal(current.contents, &request); err != nil {
			s.sendError(base.ID, -32602, "invalid shutdown params")
			return
		}
		s.enqueue(lsp.NewNullResponse(base.ID))

	case "textDocument/hover":
		var request lsp.HoverRequest
		if err := json.Unmarshal(current.contents, &request); err != nil {
			s.sendError(base.ID, -32602, "invalid hover params")
			return
		}

		response := s.state.Hover(
			request.ID,
			request.Params.TextDocument.URI,
			request.Params.Position,
		)
		if s.requestCanceled(ctx, base.ID) {
			return
		}
		s.enqueue(response)

	case "textDocument/definition":
		var request lsp.DefinitionRequest
		if err := json.Unmarshal(current.contents, &request); err != nil {
			s.sendError(base.ID, -32602, "invalid definition params")
			return
		}

		response := s.state.Definition(
			request.ID,
			request.Params.TextDocument.URI,
			request.Params.Position,
		)
		if s.requestCanceled(ctx, base.ID) {
			return
		}
		s.enqueue(response)

	case "textDocument/references":
		var request lsp.ReferenceRequest
		if err := json.Unmarshal(current.contents, &request); err != nil {
			s.sendError(base.ID, -32602, "invalid references params")
			return
		}

		response := s.state.References(
			request.ID,
			request.Params.TextDocument.URI,
			request.Params.Position,
			request.Params.Context.IncludeDeclaration,
		)
		if s.requestCanceled(ctx, base.ID) {
			return
		}
		s.enqueue(response)

	case "textDocument/documentSymbol":
		var request lsp.DocumentSymbolRequest
		if err := json.Unmarshal(current.contents, &request); err != nil {
			s.sendError(base.ID, -32602, "invalid document symbol params")
			return
		}

		response := s.state.DocumentSymbols(
			request.ID,
			request.Params.TextDocument.URI,
		)
		if s.requestCanceled(ctx, base.ID) {
			return
		}
		s.enqueue(response)

	case "textDocument/foldingRange":
		var request lsp.FoldingRangeRequest
		if err := json.Unmarshal(current.contents, &request); err != nil {
			s.sendError(base.ID, -32602, "invalid folding range params")
			return
		}

		response := s.state.FoldingRanges(
			request.ID,
			request.Params.TextDocument.URI,
		)
		if s.requestCanceled(ctx, base.ID) {
			return
		}
		s.enqueue(response)

	case "textDocument/codeAction":
		var request lsp.CodeActionRequest
		if err := json.Unmarshal(current.contents, &request); err != nil {
			s.sendError(base.ID, -32602, "invalid code action params")
			return
		}

		response := s.state.TextDocumentCodeAction(
			request.ID,
			request.Params.TextDocument.URI,
			request.Params.Range,
		)
		if s.requestCanceled(ctx, base.ID) {
			return
		}
		s.enqueue(response)

	case "textDocument/completion":
		var request lsp.CompletionRequest
		if err := json.Unmarshal(current.contents, &request); err != nil {
			s.sendError(base.ID, -32602, "invalid completion params")
			return
		}

		response := s.state.TextDocumentCompletion(
			request.ID,
			request.Params.TextDocument.URI,
			request.Params.Position,
		)
		if s.requestCanceled(ctx, base.ID) {
			return
		}
		s.enqueue(response)

	default:
		s.logger.Printf("unsupported request: %s", base.Method)
		s.sendError(base.ID, -32601, "method not found")
	}
}

func (s *Server) requestCanceled(ctx context.Context, id json.RawMessage) bool {
	if ctx.Err() == nil {
		return false
	}
	s.sendError(id, -32800, "request cancelled")
	return true
}

func (s *Server) sendError(id json.RawMessage, code int, message string) {
	s.enqueue(lsp.NewErrorResponse(id, code, message))
}

func (s *Server) enqueue(message any) {
	s.outMu.Lock()
	defer s.outMu.Unlock()
	if s.outClosed {
		return
	}
	s.out <- message
}

func (s *Server) registerCancel(key string, cancel context.CancelFunc) {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()

	if s.canceledRequests[key] {
		delete(s.canceledRequests, key)
		cancel()
	}
	s.cancels[key] = cancel
}

func (s *Server) unregisterCancel(key string) {
	s.cancelMu.Lock()
	delete(s.cancels, key)
	delete(s.canceledRequests, key)
	s.cancelMu.Unlock()
}

func (s *Server) cancelRequest(key string) {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()

	if cancel, ok := s.cancels[key]; ok {
		cancel()
		return
	}
	s.canceledRequests[key] = true
}

func (s *Server) scheduleDiagnostics(uri string) {
	s.diagnosticsMu.Lock()
	defer s.diagnosticsMu.Unlock()

	s.diagnosticsSeq++
	token := s.diagnosticsSeq
	s.diagnosticsLatest[uri] = token

	if timer, ok := s.diagnosticsTimers[uri]; ok {
		timer.Stop()
	}

	s.diagnosticsTimers[uri] = time.AfterFunc(s.diagnosticsDelay, func() {
		s.publishDiagnostics(uri, token)
	})
}

func (s *Server) publishDiagnostics(uri string, token uint64) {
	s.diagnosticsMu.Lock()
	if s.diagnosticsLatest[uri] != token {
		s.diagnosticsMu.Unlock()
		return
	}
	s.diagnosticsMu.Unlock()

	diagnostics := s.state.Diagnostics(uri)

	s.publishMu.Lock()
	defer s.publishMu.Unlock()

	s.diagnosticsMu.Lock()
	if s.diagnosticsLatest[uri] != token {
		s.diagnosticsMu.Unlock()
		return
	}
	delete(s.diagnosticsTimers, uri)
	delete(s.diagnosticsLatest, uri)
	s.diagnosticsMu.Unlock()

	s.publish(uri, diagnostics)
}

func (s *Server) clearDiagnostics(uri string) {
	s.publishMu.Lock()
	defer s.publishMu.Unlock()

	s.diagnosticsMu.Lock()
	delete(s.diagnosticsLatest, uri)
	if timer, ok := s.diagnosticsTimers[uri]; ok {
		timer.Stop()
		delete(s.diagnosticsTimers, uri)
	}
	s.diagnosticsMu.Unlock()

	s.publish(uri, []lsp.Diagnostic{})
}

func (s *Server) publish(uri string, diagnostics []lsp.Diagnostic) {
	s.enqueue(lsp.PublishDiagnosticsNotification{
		Notificaton: lsp.NewNotification("textDocument/publishDiagnostics"),
		Params: lsp.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: diagnostics,
		},
	})
}

func (s *Server) flushDiagnostics() {
	s.diagnosticsMu.Lock()
	uris := make([]string, 0, len(s.diagnosticsTimers))
	for uri, timer := range s.diagnosticsTimers {
		timer.Stop()
		uris = append(uris, uri)
	}
	s.diagnosticsTimers = map[string]*time.Timer{}
	s.diagnosticsLatest = map[string]uint64{}
	s.diagnosticsMu.Unlock()

	sort.Strings(uris)
	for _, uri := range uris {
		diagnostics := s.state.Diagnostics(uri)
		s.publishMu.Lock()
		s.publish(uri, diagnostics)
		s.publishMu.Unlock()
	}
}

func (s *Server) shutdown(shouldExit bool) {
	if shouldExit {
		s.logger.Print("client requested exit")
	}

	s.flushDiagnostics()

	close(s.jobs)
	s.workerWG.Wait()
	s.cancel()

	s.outMu.Lock()
	if !s.outClosed {
		s.outClosed = true
		close(s.out)
	}
	s.outMu.Unlock()

	s.writerWG.Wait()
}
