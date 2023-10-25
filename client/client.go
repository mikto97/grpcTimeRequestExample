package main

import (
	"bufio"
	"context"
	"flag"
	proto "grpc/grpc"
	"io"
	"log"
	"os"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	id         int
	portNumber int
}

var (
	clientPort = flag.Int("cPort", 0, "client port number")
	serverPort = flag.Int("sPort", 0, "server port number (should match the port used for the server)")
	clientID   = flag.Int("cID", 0, "client ID")
)

var ()

func main() {
	// Parse the flags to get the port for the client
	flag.Parse()

	// Create a client
	client := &Client{
		id:         *clientID,
		portNumber: *clientPort,
	}

	// Join the chat board server
	_, err := joinServer(int64(client.id))
	if err != nil {
		log.Fatalf("Could not join the chat board: %v", err)
	}

}

func getChatBoardClient() (proto.ChatBoardClient, error) {
	// Dial the server at the specified port.
	conn, err := grpc.Dial("localhost:"+strconv.Itoa(*serverPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Could not connect to port %d", *serverPort)
	} else {
		log.Printf("Connected to the chat board at port %d\n", *serverPort)
	}
	return proto.NewChatBoardClient(conn), nil
}

func joinServer(id int64) (proto.ChatBoard_JoinClient, error) {
	// Connect to the chat board
	chatBoardClient, _ := getChatBoardClient()

	// Create a bidirectional stream with the server
	stream, err := chatBoardClient.Join(context.Background())
	if err != nil {
		return nil, err
	}

	// Send the first message with the client ID
	stream.Send(&proto.JoinRequest{
		Id: int32(id),
	})

	// Receive messages from the server in a goroutine
	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				log.Println("Server closed the connection")
				return
			}
			if err != nil {
				log.Println(err)
				return
			}
			log.Println(res.GetMessage()) // print the message from the server
		}
	}()

	// Send messages to the server in a loop
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()
		// create a message with the input and send it to the server
		msg := &proto.JoinRequest{
			Message: input,
		}
		stream.Send(msg)
	}

	return stream, nil
}

/*func connectToServer() (proto.TimeAskClient, error) {
	// Dial the server at the specified port.
	conn, err := grpc.Dial("localhost:"+strconv.Itoa(*serverPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Could not connect to port %d", *serverPort)
	} else {
		log.Printf("Connected to the server at port %d\n", *serverPort)
	}
	return proto.NewTimeAskClient(conn), nil
}
*/
