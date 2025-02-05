package main

import (
	"bufio"
	"encoding/json"
	"golsp/analysis"
	"golsp/lsp"
	"golsp/rpc"
	"io"
	"log"
	"os"
)

func main() {
	logger := getLogger("/home/atharva/code/golsp/log.txt")
	logger.Println("Hey I started")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)

	state := analysis.NewState()
	writer := os.Stdout

	for scanner.Scan() {
		msg := scanner.Bytes()
		method, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			logger.Printf("Got an Error : %s", err)
			continue
		}
		haindelMessge(logger, writer, method, state, contents)
	}
}

func writeResponse(writer io.Writer, msg any) {
	reply := rpc.EncodeMessage(msg)
	writer.Write([]byte(reply))
}

func haindelMessge(
	logger *log.Logger,
	writer io.Writer,
	method string,
	state analysis.State,
	contents []byte,
) {
	logger.Printf("Recived Method : %s ", method)

	switch method {
	case "initialize":
		var requset lsp.InitializeRequest

		if err := json.Unmarshal(contents, &requset); err != nil {
			logger.Printf("hey could not parse this %s ", err)
		}

		logger.Printf(
			"Conneted to %s %s",
			requset.Params.ClientInfo.Name,
			requset.Params.ClientInfo.Version,
		)

		// hey ... let's reply!

		msg := lsp.NewInitializeResponse(requset.ID)
		writeResponse(writer, msg)
		logger.Print("Sent the reply ")

	case "textDocument/didOpen":
		var requset lsp.DidOpenTextDocumentNotifiction
		if err := json.Unmarshal(contents, &requset); err != nil {
			logger.Printf("hey could not parse this %s ", err)
		}

		logger.Printf(
			"Opened %s",
			requset.Params.TextDocument.URI,
		)

		state.OpenDocument(
			requset.Params.TextDocument.URI,
			requset.Params.TextDocument.Text,
		)

	case "textDocument/didChange":
		var request lsp.TextDocumentDidChangeNotification
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Printf("textDocument/didChange: %s", err)
			return
		}

		logger.Printf("Changed: %s", request.Params.TextDocument.URI)
		for _, change := range request.Params.ContentChanges {
			diagnostics := state.UpdateDocument(
				request.Params.TextDocument.URI,
				change.Text,
			)
			writeResponse(writer, lsp.PublishDiagnosticsNotification{
				Notificaton: lsp.Notificaton{
					RPC:    "2.0",
					Method: "textDocument/publishDiagnostics",
				},
				Params: lsp.PublishDiagnosticsParams{
					URI:         request.Params.TextDocument.URI,
					Diagnostics: diagnostics,
				},
			})
		}
	case "textDocument/hover":
		var requset lsp.HoverRequest
		if err := json.Unmarshal(contents, &requset); err != nil {
			logger.Printf("hey could not parse this %s ", err)
		}

		// create a writeResponse
		response := state.Hover(
			requset.ID,
			requset.Params.TextDocument.URI,
			requset.Params.Position,
		) // writ it back
		writeResponse(writer, response)

	case "textDocument/definition":
		var request lsp.DefinitionRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Printf("textDocument/definition: %s", err)
			return
		}

		// Create a response
		response := state.Definition(
			request.ID,
			request.Params.TextDocument.URI,
			request.Params.Position,
		)

		// Write it back
		writeResponse(writer, response)

	case "textDocument/codeAction":
		var request lsp.CodeActionRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Printf("textDocument/definition: %s", err)
			return
		}

		// Create a response
		response := state.TextDocumentCodeAction(
			request.ID,
			request.Params.TextDocument.URI,
		)

		// Write it back
		writeResponse(writer, response)

	case "textDocument/completion":
		var request lsp.CompletionRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			logger.Printf("textDocument/codeAction: %s", err)
			return
		}

		// Create a response
		response := state.TextDocumentCompletion(
			request.ID,
			request.Params.TextDocument.URI,
		)

		// Write it back
		writeResponse(writer, response)

	}
}

func getLogger(filenmae string) *log.Logger {
	logfile, err := os.OpenFile(
		filenmae,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		panic("you didnt give me s good file")
	}

	return log.New(logfile, "[golsp]", log.Ldate|log.Ltime|log.Lshortfile)
}
