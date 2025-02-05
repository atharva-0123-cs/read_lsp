package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"golsp/analysis"
	"golsp/lsp"
	"golsp/rpc"
	"log"
	"os"
)

func main() {
	fmt.Println("Start LSP")

	logger := getLogger("/home/atharva/code/golsp/log.txt")
	logger.Println("Hey I started")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)

	state := analysis.NewState()

	for scanner.Scan() {
		msg := scanner.Bytes()
		method, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			logger.Printf("Got an Error : %s", err)
			continue
		}
		haindelMessge(logger, method, state, contents)
	}
}

func haindelMessge(
	logger *log.Logger,
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
		reply := rpc.EncodeMessage(msg)

		writer := os.Stdout
		writer.Write([]byte(reply))

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
		var requset lsp.TextDocumentDidChangeNotification
		if err := json.Unmarshal(contents, &requset); err != nil {
			logger.Printf("hey could not parse this %s ", err)
		}

		logger.Printf(
			"Changed %s %s",
			requset.Params.TextDocument.URI,
			requset.Params.ContentChanges,
		)

		for _, change := range requset.Params.ContentChanges {
			state.UpdateDocument(requset.Params.TextDocument.URI, change.Text)
		}

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
