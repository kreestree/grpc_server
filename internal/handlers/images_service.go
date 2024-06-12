package handlers

import (
	"context"
	"fmt"
	"github.com/djherbis/times"
	"github.com/kreestree/grpc_server/proto/images_service"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"os"
	"path/filepath"
	"time"
)

type ImageServiceServer struct {
	images_service.UnimplementedImagesServiceServer
	getUploadImageSemaphore *semaphore.Weighted
	getImageListSemaphore   *semaphore.Weighted
	imageDir                string
	imageExt                string
}

func NewImageServiceServer(
	maxGetUploadImageConnections, maxGetImageListConnections int64,
	imageDir, imageExt string,
) *ImageServiceServer {
	return &ImageServiceServer{
		getUploadImageSemaphore: semaphore.NewWeighted(maxGetUploadImageConnections),
		getImageListSemaphore:   semaphore.NewWeighted(maxGetImageListConnections),
		imageDir:                imageDir,
		imageExt:                imageExt,
	}
}

// UploadImage saves the image in the specified directory
func (s *ImageServiceServer) UploadImage(
	ctx context.Context,
	req *images_service.UploadImageRequest,
) (*images_service.UploadImageResponse, error) {
	log.Printf("upload image request: %v", req)
	if err := s.getUploadImageSemaphore.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer s.getUploadImageSemaphore.Release(1)

	imageName := req.GetImageName()
	imageData := req.GetImageData()
	fullFileName, err := s.saveImageToDisk(imageName, imageData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error saving image: %v", err)
	}
	return &images_service.UploadImageResponse{ImageName: fullFileName}, nil
}

// GetImageList returns information about saved images
func (s *ImageServiceServer) GetImageList(
	ctx context.Context,
	req *images_service.GetImageListRequest,
) (*images_service.GetImageListResponse, error) {
	log.Printf("get image list request: %v", req)
	if err := s.getImageListSemaphore.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer s.getImageListSemaphore.Release(1)

	imagesInfo, err := s.getImagesInfo()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error reading image list: %v", err)
	}
	return &images_service.GetImageListResponse{ImageInfo: imagesInfo}, nil
}

// GetImage returns single image data
func (s *ImageServiceServer) GetImage(
	ctx context.Context,
	req *images_service.GetImageRequest,
) (*images_service.GetImageResponse, error) {
	log.Printf("get single image data request: %v", req)
	if err := s.getUploadImageSemaphore.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer s.getUploadImageSemaphore.Release(1)

	imageName := req.GetImageName()
	imageData, err := s.readImage(imageName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, status.Errorf(codes.NotFound, "Image %s not found", imageName)
		}
		return nil, status.Errorf(codes.Internal, "Error reading image: %v", err)
	}

	return &images_service.GetImageResponse{ImageData: imageData}, nil
}

// getFileName returns full file name (name + extension)
func (s *ImageServiceServer) getFileName(imageName string) string {
	return fmt.Sprintf("%s.%s", imageName, s.imageExt)
}

// saveImageToDisk performs file saving
func (s *ImageServiceServer) saveImageToDisk(imageName string, imageData []byte) (string, error) {
	fullFileName := s.getFileName(imageName)
	filePath := filepath.Join(s.imageDir, fullFileName)
	err := os.WriteFile(filePath, imageData, 0644)
	if err != nil {
		return "", err
	}
	return fullFileName, nil
}

// getCreationModificationDate returns file creation and modification dates
func (s *ImageServiceServer) getCreationModificationDate(fileName string) (string, string, error) {
	fileOpen, _ := os.Open(filepath.Join(s.imageDir, fileName))
	defer fileOpen.Close()
	timeStat, err := times.StatFile(fileOpen)
	if err != nil {
		return "", "", err
	}
	var creationDate string
	if timeStat.HasBirthTime() {
		creationDate = timeStat.BirthTime().Format(time.DateTime)
	}
	modificationDate := timeStat.ChangeTime().Format(time.DateTime)
	return creationDate, modificationDate, nil
}

// getImagesInfo returns information about saved images
func (s *ImageServiceServer) getImagesInfo() ([]*images_service.ImageInfo, error) {
	files, err := os.ReadDir(s.imageDir)
	if err != nil {
		return nil, err
	}

	var imageInfoList []*images_service.ImageInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		creationDate, modificationDate, _ := s.getCreationModificationDate(file.Name())
		imageInfo := &images_service.ImageInfo{
			ImageName:        file.Name(),
			CreationDate:     creationDate,
			ModificationDate: modificationDate,
		}
		imageInfoList = append(imageInfoList, imageInfo)
	}

	return imageInfoList, nil
}

// readImage returns image data
func (s *ImageServiceServer) readImage(imageName string) ([]byte, error) {
	imageData, err := os.ReadFile(fmt.Sprintf("%s/%s", s.imageDir, imageName))
	if err != nil {
		return nil, err
	}
	return imageData, nil
}
