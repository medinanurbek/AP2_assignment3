package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	orderpb "github.com/medinanurbek/generated-repo/go/order"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	orderID := flag.String("order_id", "", "ID of the order to subscribe to")
	serverAddr := flag.String("server", "localhost:50052", "The server address in the format of host:port")
	flag.Parse()

	if *orderID == "" {
		log.Fatal("Error: order_id is required. Use --order_id=YOUR_UUID")
	}

	// 1. Establish connection to the Order gRPC server
	conn, err := grpc.NewClient(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Couldn't connect to %s: %v", *serverAddr, err)
	}
	defer conn.Close()

	client := orderpb.NewOrderServiceClient(conn)

	// 2. Call the server-side streaming method
	fmt.Printf("[Client] Subscribing to updates for Order: %s ...\n", *orderID)
	stream, err := client.SubscribeToOrderUpdates(context.Background(), &orderpb.OrderRequest{
		OrderId: *orderID,
	})
	if err != nil {
		log.Fatalf("Error calling SubscribeToOrderUpdates: %v", err)
	}

	// 3. Receive and print updates from the stream
	for {
		update, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("[Client] Stream closed by server.")
			break
		}
		if err != nil {
			log.Fatalf("[Client] Error receiving from stream: %v", err)
		}

		// Convert protobuf timestamp to human readable local time
		updatedTime := update.GetUpdatedAt().AsTime().Local().Format(time.RFC1123)

		fmt.Printf("\n--- NEW UPDATE RECEIVED ---\n")
		fmt.Printf("Order ID: %s\n", update.GetOrderId())
		fmt.Printf("Status:   %s\n", update.GetStatus())
		fmt.Printf("Time:     %s\n", updatedTime)
		fmt.Printf("---------------------------\n")
	}
}
