package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/hyuko21/file-storage-s3-golang/internal/auth"
	"github.com/hyuko21/file-storage-s3-golang/internal/utils"
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

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't find video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusForbidden, "Can't upload a thumbnail for this video", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)
	tnURL, err := cfg.uploadThumbnail(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to save uploaded file as thumbnail", err)
		return
	}

	video.ThumbnailURL = &tnURL
	cfg.db.UpdateVideo(video)

	respondWithJSON(w, http.StatusOK, video)
}

func (cfg *apiConfig) uploadThumbnail(r *http.Request) (tnURL string, err error) {
	const maxMemory = 10 << 20 // 10mb (as in 10 * 1024 * 1024, as 1024 is 1 << 10)
	r.ParseMultipartForm(maxMemory)

	file, fHeader, err := r.FormFile("thumbnail")
	if err != nil {
		return
	}
	defer file.Close()

	mediaTypeParts, err := parseThumbnailMediaType(fHeader.Header.Get("Content-Type"))
	if err != nil {
		return
	}
	fileExt := mediaTypeParts[1]
	tnFileKey, err := utils.MakeRandomString()
	if err != nil {
		return
	}
	tnFilename := fmt.Sprintf("%s.%s", tnFileKey, fileExt)
	tnFilepath := filepath.Join(cfg.assetsRoot, tnFilename)
	tnFile, err := os.Create(tnFilepath)
	if err != nil {
		return
	}
	fBytes, err := io.ReadAll(file)
	if err != nil {
		return
	}
	_, err = tnFile.Write(fBytes)
	if err != nil {
		return
	}

	assetsRootPath := strings.TrimPrefix(cfg.assetsRoot, "./")
	tnURL = fmt.Sprintf("http://localhost:8091/%s/%s", assetsRootPath, tnFilename)
	return
}

func parseThumbnailMediaType(contentType string) (mediaTypeParts []string, err error) {
	fMediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return
	}
	mediaTypeParts = strings.Split(fMediaType, "/")
	if mediaTypeParts[0] != "image" || mediaTypeParts[1] != "jpeg" && mediaTypeParts[1] != "png" {
		return nil, fmt.Errorf("unsupported media type for thumbnail: %s", mediaTypeParts[0])
	}
	return
}
