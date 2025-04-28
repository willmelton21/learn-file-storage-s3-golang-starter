package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	_"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
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

	// TODO: implement the upload here
	maxMemory := 10 << 20

	err = r.ParseMultipartForm(int64(maxMemory))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error with MultipartForm Parsing",err)
		return
	}

	file,header, err := r.FormFile("thumbnail")
	defer file.Close()

	mediaTypeFromHeader := header.Header.Get("Content-Type")

	stream, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't Read file to stream",err)
		return
	}

	dbVid, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video from database",err)
		return
   }
	if dbVid.UserID !=  userID {
		respondWithError(w, http.StatusUnauthorized, "User ID does not match database ID for video",err)
		return
	}

	newThumbnail := thumbnail{data: stream,mediaType: mediaTypeFromHeader}

	videoThumbnails[videoID] = newThumbnail
   thumbnailURL := fmt.Sprintf("http://localhost:%d/api/thumbnails/%d", cfg.port, videoID)
	dbVid.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(dbVid)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update file with new URL",err)
	}

	
	respondWithJSON(w, http.StatusOK, dbVid)
}
