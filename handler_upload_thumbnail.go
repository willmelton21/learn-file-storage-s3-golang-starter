package main

import (
	"fmt"
	"io"
	"net/http"
   "os"
	"mime"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	_ "github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
	"path/filepath"
	"crypto/rand"
	"encoding/base64"
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
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error forming file from thumbnail",err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	extension := ""
	switch mediaType {
    case "image/png":
        extension = "png"
    case "image/jpeg":
        extension = "jpg"
    // Add other cases as needed
    default:
        extension = "bin" // Fallback
}
	parsedMediaType,_,err :=	mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't Parse Media Type",err)
		return
	}
	fmt.Println("parsedMediaType is ",parsedMediaType)
	if parsedMediaType != "image/jpeg" && parsedMediaType != "image/png" {
		respondWithError(w, http.StatusUnauthorized, "Only png and jpegs can be used for Upload",err)
		return
	}


	key := make([]byte,32)
	bytesRead, err :=	rand.Read(key)
	if bytesRead != len(key) {
		respondWithError(w, http.StatusInternalServerError, "BytesRead was not equal to the length of buffer",err)
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fill buffer with data",err)
		return
	}

	urlEncoded := base64.RawURLEncoding.EncodeToString(key)


	filePath := filepath.Join(cfg.assetsRoot,urlEncoded +"." + extension) 
	filePathFull := "http://localhost:" + cfg.port + "/" + filePath 

	createdFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create file with given file string",err)
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

	_, err = io.Copy(createdFile,file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy file to assets directory",err)
		return

	}

	//encodedString := base64.StdEncoding.EncodeToString(stream)

	//dataURL := fmt.Sprintf("data:%s;base64,%s",mediaTypeFromHeader,encodedString)

	//newThumbnail := thumbnail{data: stream,mediaType: mediaTypeFromHeader}

   //thumbnailURL := fmt.Sprintf("http://localhost:%s/api/thumbnails/%d", cfg.port, videoID)
	dbVid.ThumbnailURL = &filePathFull
	err = cfg.db.UpdateVideo(dbVid)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update file with new URL",err)
	}

	
	respondWithJSON(w, http.StatusOK, dbVid)
}
