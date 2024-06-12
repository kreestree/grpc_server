package main

import (
	"github.com/kreestree/grpc_server/internal/handlers"
	"github.com/kreestree/grpc_server/proto/images_service"
	"google.golang.org/grpc"
	"log"
	"net"
)

const (
	maxGetUploadImageConnections = 10       // Maximum number of simultaneous connections for UploadImage and GetImage
	maxGetImageListConnections   = 100      // Maximum number of simultaneous connections for GetImageList
	imageDir                     = "media/" // Images directory
	imageExt                     = "jpg"    // Images extension
)

func main() {
	listener, err := net.Listen("tcp", ":8089")
	if err != nil {
		log.Fatalf("failed to start listener: %v", err)
	}

	server := grpc.NewServer()
	imageService := handlers.NewImageServiceServer(
		maxGetUploadImageConnections,
		maxGetImageListConnections,
		imageDir,
		imageExt,
	)
	images_service.RegisterImagesServiceServer(server, imageService)
	err = server.Serve(listener)
	if err != nil {
		log.Fatalf("cannot serve: %v", err)
	}
}
