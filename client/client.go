package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	proto "grpc/grpc"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

func getServerAddress() string {
	mainServer := "localhost:50051"
	replicaManager := "localhost:50040"

	// Try to connect to the main server
	conn, err := grpc.Dial(mainServer, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
	if err == nil {
		conn.Close() // Close the connection if successful
		return mainServer
	}

	// If main server is down, return the replica manager's address
	return replicaManager
}

func main() {
	flag.Parse()
	BidderID := flag.Arg(0)
	port, err := strconv.Atoi(flag.Arg(1))
	if err != nil {
		log.Fatal("Invalid port number:", err)
	}

	leaderAddress := getServerAddress()
	if leaderAddress == "" {
		log.Fatal("No leader address available, exiting")
	}

	conn, err := grpc.Dial(leaderAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer conn.Close()

	client := proto.NewAuctionClient(conn)

	fmt.Printf("%s connected at port: %d\n", BidderID, port)

	fmt.Printf("Auction is open. You can bid until %s.\n", time.Now().Add(30*time.Second).Format("15:04:05"))

	for {
		requestID := generateUniqueRequestID()

		var bidAmount int
		fmt.Print("Enter your bid amount (0 to exit): ")
		fmt.Scan(&bidAmount)

		if bidAmount == 0 {
			break
		}

		bidRequest := &proto.BidRequest{
			RequestId: requestID,
			BidderId:  BidderID,
			Amount:    int32(bidAmount),
		}

		// Send the bid to the main server
		_, err = client.Bid(context.Background(), bidRequest)
		if err != nil {
			fmt.Println("Error sending bid to main server:", err)
			// Optionally handle the error, e.g., retrying
		}

		// Replicate the bid to the replica manager
		sendAndReplicateBid(bidRequest)

	}

	resultRequest := &proto.ResultRequest{RequestId: generateUniqueRequestID()}
	resultResponse, err := client.Result(context.Background(), resultRequest)
	if err != nil {
		log.Fatalf("Result failed: %v", err)
	}
	fmt.Println("Result Outcome:", resultResponse.Outcome)
}

func generateUniqueRequestID() string {
	return uuid.New().String()
}

func sendAndReplicateBid(bidRequest *proto.BidRequest) {
	mainServerAddress := "localhost:50051"
	replicaManagerAddress := "localhost:50040"

	// Try sending the bid to the main server
	if err := sendBidToServer(mainServerAddress, bidRequest, false); err != nil {
		fmt.Println("Error sending bid to main server, switching to replica manager...")
		// Switch to the replica manager if the main server is down
		sendBidToServer(replicaManagerAddress, bidRequest, true)
	} else {
		// If bid to the main server was successful, replicate it to the replica manager
		sendBidToServer(replicaManagerAddress, bidRequest, true)
	}
}

func sendBidToServer(serverAddress string, bidRequest *proto.BidRequest, isReplica bool) error {
	conn, err := grpc.Dial(serverAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("Failed to connect to server at %s: %v", serverAddress, err)
		return err
	}
	defer conn.Close()

	client := proto.NewAuctionClient(conn)

	if isReplica {
		// Call ReplicateBidRequest for the replica manager
		_, err = client.ReplicateBidRequest(context.Background(), bidRequest)
	} else {
		// Call Bid for the main server
		_, err = client.Bid(context.Background(), bidRequest)
	}

	if err != nil {
		log.Printf("Failed to send bid to server at %s: %v", serverAddress, err)
		return err
	} else {
		fmt.Printf("Bid successfully sent to server at %s.\n", serverAddress)
	}
	return nil
}
