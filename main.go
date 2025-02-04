package main

import (
	"bufio"
	"encoding/json"
	"fmt"
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

	for scanner.Scan() {
		msg := scanner.Bytes()
		method, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			logger.Printf("Got an Error : %s", err)
			continue
		}
		haindelMessge(logger, method, contents)
	}
}

func haindelMessge(logger *log.Logger, method string, contents []byte) {
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
