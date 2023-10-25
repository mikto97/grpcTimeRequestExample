package main

import (
	"flag"
	"fmt"
	proto "grpc/grpc"
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"google.golang.org/grpc"
)

// Struct that will be used to represent the Server.
type Server struct {
	proto.UnimplementedTimeAskServer // Necessary
	proto.UnimplementedChatBoardServer
	name string
	port int
}

// Used to get the user-defined port for the server from the command line
var port = flag.Int("port", 0, "server port number")

func main() {
	// Get the port from the command line when the server is run
	flag.Parse()

	// Create a server struct
	server := &Server{
		name: "smallPigsServer",
		port: *port,
	}

	// Start the server
	go startServer(server)

	// Keep the server running until it is manually quit
	for {

	}
}

func startServer(server *Server) {

	// Create a new grpc server
	grpcServer := grpc.NewServer()

	// Create a service implementation
	chatBoardServer := &ChatBoardServer{
		clients: make(map[int32]proto.ChatBoard_JoinServer),
	}

	//Make the server listen at the given port (convert int port to string)
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(server.port))

	if err != nil {
		log.Fatalf("Could not create the server %v", err)
	}
	log.Printf("test - Started server at port: %d\n", server.port)

	// Register the grpc server and serve its listener
	//proto.RegisterTimeAskServer(grpcServer, chatBoardServer) needed to remove this for it to work
	proto.RegisterChatBoardServer(grpcServer, chatBoardServer)
	serveError := grpcServer.Serve(listener)
	if serveError != nil {
		log.Fatalf("Could not serve listener")
	}
	log.Print(" server is now listening")

}

type ChatBoardServer struct {
	proto.UnimplementedChatBoardServer
	clients map[int32]proto.ChatBoard_JoinServer // a map of client IDs and streams
	mu      sync.Mutex                           // a mutex for locking the map
}

func (s *ChatBoardServer) Join(stream proto.ChatBoard_JoinServer) error {
	// receive the first message from the client, which contains its ID
	req, err := stream.Recv()
	if err != nil {
		return err
	}
	id := req.GetId() // get the client ID

	// add a defer statement to handle stream errors
	defer func() {
		log.Printf("Client %d has left the chat board.", id)
		// the client has closed the connection, remove it from the map
		s.mu.Lock()
		delete(s.clients, id)
		s.mu.Unlock()

		// broadcast a message to all other clients that the client has left
		s.broadcast(fmt.Sprintf("Client %d has left the chat board.", id), id)
		//}
	}()

	// add the client stream to the map
	s.mu.Lock()
	s.clients[id] = stream
	s.mu.Unlock()
	log.Printf("Client %d has joined the chat board.", id)
	// send a welcome message to the new client
	stream.Send(&proto.JoinResponse{
		Message: fmt.Sprintf("Welcome, client %d!", id),
	})

	// broadcast a message to all other clients that a new client has joined
	s.broadcast(fmt.Sprintf("Client %d has joined the chat board.", id), id)

	// wait for any further messages from the client or stream errors
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			// the client has closed the connection, remove it from the map
			s.mu.Lock()
			delete(s.clients, id)
			s.mu.Unlock()

			// broadcast a message to all other clients that the client has left
			s.broadcast(fmt.Sprintf("Client %d has left the chat board.", id), id)
			return nil
		}
		if err != nil {
			return err
		}
		// handle any other messages from the client here
		// for example, you can print the message to the server log
		log.Printf("Received message from client %d: %s\n", id, msg.GetMessage())
		// or you can send a response back to the client
		stream.Send(&proto.JoinResponse{
			Message: fmt.Sprintf("Server received your message: %s", msg.GetMessage()),
		})
		// or you can broadcast the message to other clients
		s.broadcast(fmt.Sprintf("Client %d says: %s", id, msg.GetMessage()), id)
	}
}

// broadcast sends a message to all clients except the sender
func (s *ChatBoardServer) broadcast(message string, sender int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, stream := range s.clients {
		if id == sender {
			continue // skip the sender
		}
		stream.Send(&proto.JoinResponse{
			Message: message,
		})
	}
}
