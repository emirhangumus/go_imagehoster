package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/buckket/go-blurhash"
	"github.com/golang-jwt/jwt"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {

	// check is token valid
	tokenString := r.Header.Get("Authorization")
	tokenString = strings.Replace(tokenString, "Bearer ", "", 1)
	claims := jwt.MapClaims{}
	_, jwterror := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(goDotEnvVariable("JWT_KEY")), nil
	})
	if jwterror != nil {
		http.Error(w, `{"success": false, "error": "Invalid token"}`, http.StatusUnauthorized)
		return
	}

	// Parse the form data with a maximum of 10 MB in memory
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(
			w,
			`{"success": false, "error": "Unable to parse the form data. Please try again."}`,
			http.StatusBadRequest)
		return
	}

	// Retrieve the file from the form data
	file, handler, err := r.FormFile("image")
	if err != nil {
		http.Error(w, `{"success": false, "error": "Unable to retrieve the file from the form data"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	newFilename := generateNewFilename(handler.Filename)

	// Create a destination file in the folder where you want to save the image
	// You can customize the folder and filename as needed
	destinationPath := "./uploads/" + newFilename
	dst, err := os.Create(destinationPath)
	if err != nil {
		http.Error(w, `{"success": false, "error": "Unable to create the file for the uploaded image. Check the write permissions on the upload folder."}`, http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file data to the destination file
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, `{"success": false, "error": "Unable to copy the uploaded file data to the destination file on the server"}`, http.StatusInternalServerError)
		return
	}

	// Open uploaded file
	uploadedFile, err := os.Open(destinationPath)
	
	if err != nil {
		http.Error(w, `{"success": false, "error": "Unable to open the uploaded file"}`, http.StatusInternalServerError)
		return
	}

	defer uploadedFile.Close()

	// get the extension of uploaded file
	extension := filepath.Ext(destinationPath)

	// decode the uploaded file into image.Image
	var img image.Image
	if extension == ".png" {
		img, err = png.Decode(uploadedFile)
	}
	if extension == ".jpg" || extension == ".jpeg" {
		img, err = jpeg.Decode(uploadedFile)
	}

	if err != nil {
		http.Error(w, `{"success": false, "error": "Unable to decode the uploaded file into image.Image"}`, http.StatusInternalServerError)
		return
	}

	// encode the image.Image into blurhash
	hash, err := blurhash.Encode(4, 3, img)

	if err != nil {
		http.Error(w, `{"success": false, "error": "Unable to encode the image into blurhash"}`, http.StatusInternalServerError)
		return
	}

	// return the blurhash and path of the uploaded file
	/*
		{
			"success": true,
			"data": {
				"path": "uploads/image_1629780000_00000001.png",
				"blurhash: "LdGc0c00P*00?w00R*00?w00R*00"
			}
		}
	*/
	// remove first 2 characters from the path
	destinationPath = destinationPath[2:]
	fmt.Fprintf(w, `{"success": true, "data": {"path": "%s", "blurhash": "%s"}}`, destinationPath, hash)
}

func generateNewFilename(originalFilename string) string {
	// Get the file extension from the original filename
	ext := filepath.Ext(originalFilename)

	// Initialize the random number generator with a unique source
	source := rand.NewSource(time.Now().UnixNano())
	randomNumber := rand.New(source).Intn(100000000)

	// Get the current date-time in seconds
	currentTimeInSeconds := time.Now().Unix()

	// Format the new filename
	newFilename := fmt.Sprintf("image_%d_%08d%s", currentTimeInSeconds, randomNumber, ext)

	return newFilename
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the filename from the URL path
	fileName := strings.TrimPrefix(r.URL.Path, "/uploads/")

	if len(fileName) == 0 {
		http.NotFound(w, r)
		return
	}

	// Construct the full path to the file
	filePath := "./uploads/" + fileName

	// Check if the file exists
	_, err := os.Stat(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Serve the file
	http.ServeFile(w, r, filePath)
}

// use godot package to load/read the .env file and
// return the value of the key
func goDotEnvVariable(key string) string {

	// load .env file
	err := godotenv.Load(".env")
  
	if err != nil {
		fmt.Println("Error loading .env file")
	}
  
	return os.Getenv(key)
  }
  

// allow for port 3000
const allowOriginsList = "http://localhost:3000"

func main() {
	godotenv.Load(".env")
	PORT := goDotEnvVariable("PORT")
	println(PORT)
	// Create a new CORS handler with the desired options
	corsHandler := cors.New(cors.Options{
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"}, // Allow all headers
		MaxAge:         86400,         // Maximum cache age (1 day)
		AllowOriginFunc: func(origin string) bool {
			if origin == "" || strings.Contains(allowOriginsList, origin) {
				return true
			}
			return false
		},
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"success": true, "message": "Welcome to Image Hoster API written in Go by Emirhan Gümüş"}`)
	})

	// serve static files
	http.HandleFunc("/uploads/", fileHandler)
	
	// Wrap your HTTP handler with the CORS handler
	http.Handle("/upload", corsHandler.Handler(http.HandlerFunc(uploadHandler)))
	fmt.Println("Server is running on port " + PORT)
	http.ListenAndServe(":"+PORT, nil)
}
