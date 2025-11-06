package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/hyuko21/file-storage-s3-golang/internal/auth"
	"github.com/hyuko21/file-storage-s3-golang/internal/utils"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find video draft", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusForbidden, "Can't upload video file to this video draft", err)
		return
	}

	fmt.Println("uploading video file to video draft", videoID, "by user", userID)
	videoURL, err := cfg.uploadVideo(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to save uploaded video file", err)
		return
	}

	video.VideoURL = &videoURL
	cfg.db.UpdateVideo(video)

	respondWithJSON(w, http.StatusOK, video)
}

func (cfg *apiConfig) uploadVideo(r *http.Request) (videoURL string, err error) {
	const maxMemory = 1 << 30 // 1gb
	r.ParseMultipartForm(maxMemory)

	uploadedFile, fHeader, err := r.FormFile("video")
	if err != nil {
		return
	}
	defer uploadedFile.Close()

	fContentType := fHeader.Header.Get("Content-Type")
	mediaTypeParts, err := parseVideoMediaType(fContentType)
	if err != nil {
		return
	}
	fileExt := mediaTypeParts[1]
	if err != nil {
		return
	}
	randomFileKey, err := utils.MakeRandomString()
	if err != nil {
		return
	}
	tempVideoPattern := fmt.Sprintf("%s.%s", "tubely-golang-video", fileExt)
	tempVideoFile, err := os.CreateTemp("", tempVideoPattern)
	if err != nil {
		return
	}
	defer os.Remove(tempVideoFile.Name())
	defer tempVideoFile.Close()

	_, err = io.Copy(tempVideoFile, uploadedFile)
	if err != nil {
		return
	}
	aspectRatio, err := utils.GetVideoAspectRatio(tempVideoFile.Name())
	if err != nil {
		return
	}
	_, err = tempVideoFile.Seek(0, io.SeekStart)
	if err != nil {
		return
	}

	processedFilePath, err := utils.ProcessVideoForFastStart(tempVideoFile.Name())
	if err != nil {
		return
	}
	processedTempVideoFile, err := os.OpenFile(processedFilePath, os.O_RDONLY, 0444)
	if err != nil {
		return
	}
	defer os.Remove(processedTempVideoFile.Name())
	defer processedTempVideoFile.Close()

	fileKey := fmt.Sprintf("%s.%s", randomFileKey, fileExt)
	switch aspectRatio {
	case "16:9":
		fileKey = "landscape/" + fileKey
	case "9:16":
		fileKey = "portrait/" + fileKey
	case "other":
		fileKey = "other/" + fileKey
	}
	cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileKey,
		Body:        processedTempVideoFile,
		ContentType: &fContentType,
	})

	videoURL = fmt.Sprintf("%s/%s", cfg.s3CfDistribution, fileKey)
	return
}

func parseVideoMediaType(contentType string) (mediaTypeParts []string, err error) {
	fMediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return
	}
	mediaTypeParts = strings.Split(fMediaType, "/")
	if mediaTypeParts[0] != "video" || mediaTypeParts[1] != "mp4" {
		return nil, fmt.Errorf("unsupported media type for thumbnail: %s", mediaTypeParts[0])
	}
	return
}
