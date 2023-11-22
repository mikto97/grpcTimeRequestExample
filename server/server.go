package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	proto "grpc/grpc"

	"google.golang.org/grpc"
)

type AuctionServer struct {
	Bids       map[string]int32 // map[bidderID]bidAmount
	Lock       sync.Mutex
	AuctionEnd time.Time
	proto.UnimplementedAuctionServer
}

func NewAuctionServer() *AuctionServer {
	return &AuctionServer{
		Bids: make(map[string]int32),
	}
}

func (s *AuctionServer) Bid(ctx context.Context, req *proto.BidRequest) (*proto.BidResponse, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	// Check if the auction has ended
	if time.Now().After(s.AuctionEnd) {
		return &proto.BidResponse{Outcome: "Auction has ended. Bids are no longer accepted."}, nil
	}

	// Update the bid if it's higher than the previous one
	currentBid, ok := s.Bids[req.BidderId]
	if !ok || req.Amount > currentBid {
		s.Bids[req.BidderId] = req.Amount
		fmt.Printf("Bidder: %s has the highest bid with: %d at %s", req.BidderId, req.Amount, time.Now())
		return &proto.BidResponse{Outcome: "Bid accepted."}, nil
	}

	return &proto.BidResponse{Outcome: "Bid rejected. The bid amount must be higher than the current highest bid."}, nil
}

func (s *AuctionServer) Result(ctx context.Context, req *proto.ResultRequest) (*proto.ResultResponse, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	// Check if the auction has ended
	if time.Now().Before(s.AuctionEnd) {
		fmt.Printf("Auction is still ongoing. Result not available yet.")
		return &proto.ResultResponse{Outcome: "Auction is still ongoing. Result not available yet."}, nil
	}

	// Determine the winner
	var winner string
	var highestBid int32
	for bidder, bid := range s.Bids {
		if bid > highestBid {
			highestBid = bid
			winner = bidder
		}
	}
	fmt.Printf("Winner: Bidder %s with amount %d", winner, highestBid)

	return &proto.ResultResponse{Outcome: fmt.Sprintf("Winner: Bidder %s with amount %d", winner, highestBid)}, nil
}

func main() {
	/****MAIN SERVER LOGIC****/
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	mainServer := grpc.NewServer()
	auctionServer := NewAuctionServer()
	log.Printf("Server created. Listening on port %s", lis.Addr().String())

	// Set the auction end time (e.g., 100 seconds from now)
	auctionServer.AuctionEnd = time.Now().Add(100 * time.Second)

	proto.RegisterAuctionServer(mainServer, auctionServer)

	go func() {
		if err := mainServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	select {}
}
