package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	_"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"crypto/rand"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	r.Body =	http.MaxBytesReader(w,r.Body,1 << 30)
	videoIDStr := r.URL.String()
   videoIDStr = videoIDStr[18:]
	fmt.Println("videoID is ",videoIDStr)
	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't Parse Video ID", err)
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
		respondWithError(w, http.StatusNotFound, "Couldn't get video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Video does not belong to authenticated user", err)
		return
	}
	// Set a 1 GB upload limit (1 << 30 bytes)
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

	// Then parse the multipart form
	err = r.ParseMultipartForm(32 << 20) // This sets a buffer size for form parsing, not the total limit
	if err != nil {
   	 respondWithError(w, http.StatusBadRequest, "Failed to parse form", err)
    	return
	}

	file,header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error forming file from thumbnail",err)
		return
	}
	defer file.Close()


	mediaType := header.Header.Get("Content-Type")

	mediatype, _, err :=	mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing video",err)
		return
	}
	
	if mediatype != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "parsed media type was not video/mp4 ",err)
		return
	}

	tmpfile, err := os.CreateTemp("","tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create tmp file ",err)
		return
	}

	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	_, err = io.Copy(tmpfile,file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy to tmp file",err)
		return
	}

	tmpfile.Seek(0, io.SeekStart)	

	randomByte := make([]byte, 16)
	rand.Read(randomByte)
	key := fmt.Sprintf("%x.mp4",randomByte)

	s3Struct := s3.PutObjectInput{Bucket: &cfg.s3Bucket,
							Key:  &key,
							Body: tmpfile,
							ContentType: &mediaType}
	
	_, err = cfg.S3Client.PutObject(r.Context(),&s3Struct)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't put object in s3 bucket",err)
		return
	}

	newURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",cfg.s3Bucket,cfg.s3Region,key)

	video.VideoURL = &newURL

	 err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update database video",err)
		return
	}

}
