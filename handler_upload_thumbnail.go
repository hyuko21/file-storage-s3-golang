package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/hyuko21/file-storage-s3-golang/internal/auth"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20 // 10mb (as in 10 * 1024 * 1024, as 1024 is 1 << 10)
	r.ParseMultipartForm(maxMemory)

	file, fHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	fBytes, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to read file contents", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Can't upload a thumbnail for this video", err)
		return
	}

	fContentType := fHeader.Header.Get("Content-Type")
	contentTypeParts := strings.Split(fContentType, "/")
	if len(contentTypeParts) < 2 || contentTypeParts[0] != "image" {
		respondWithError(w, http.StatusBadRequest, "Unable to save this file as thumbnail", errors.New("unknown file media type"))
		return
	}
	fileExt := contentTypeParts[1]
	tnFilename := fmt.Sprintf("%s.%s", videoID, fileExt)
	tnFilepath := filepath.Join(cfg.assetsRoot, tnFilename)
	tnFile, err := os.Create(tnFilepath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to save this file as thumbnail", err)
		return
	}
	_, err = tnFile.Write(fBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save this file", err)
		return
	}

	assetsRootPath := strings.TrimPrefix(cfg.assetsRoot, "./")
	thumbnailURL := fmt.Sprintf("http://localhost:8091/%s/%s", assetsRootPath, tnFilename)
	video.ThumbnailURL = &thumbnailURL
	cfg.db.UpdateVideo(video)

	respondWithJSON(w, http.StatusOK, video)
}
