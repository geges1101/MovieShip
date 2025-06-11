package internal

import (
	"net/http"
	"strconv"
	"time"

	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

type Handlers struct {
	DB *gorm.DB
}

func (h *Handlers) ListMovies(c *gin.Context) {
	var movies []Movie
	if err := h.DB.Find(&movies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	c.JSON(http.StatusOK, movies)
}

func (h *Handlers) GetMovie(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var movie Movie
	if err := h.DB.First(&movie, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, movie)
}

func (h *Handlers) GetWatchHistory(c *gin.Context) {
	email, _ := c.Get("user_email")
	var user User
	if err := h.DB.Where("email = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	var history []WatchHistory
	if err := h.DB.Where("user_id = ?", user.ID).Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	c.JSON(http.StatusOK, history)
}

func (h *Handlers) UpdateWatchHistory(c *gin.Context) {
	email, _ := c.Get("user_email")
	var user User
	if err := h.DB.Where("email = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	var input struct {
		MovieID  uint `json:"movie_id"`
		Progress int  `json:"progress"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	var wh WatchHistory
	if err := h.DB.Where("user_id = ? AND movie_id = ?", user.ID, input.MovieID).First(&wh).Error; err != nil {
		wh = WatchHistory{UserID: user.ID, MovieID: input.MovieID, Progress: input.Progress, LastWatch: time.Now()}
		h.DB.Create(&wh)
	} else {
		wh.Progress = input.Progress
		wh.LastWatch = time.Now()
		h.DB.Save(&wh)
	}
	c.JSON(http.StatusOK, wh)
}

func (h *Handlers) UploadVideo(c *gin.Context) {
	roles, _ := c.Get("user_roles")
	isAdmin := false
	for _, r := range roles.([]string) {
		if r == "admin" {
			isAdmin = true
		}
	}
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
		return
	}
	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	title := c.PostForm("title")
	desc := c.PostForm("description")
	minioClient, err := NewMinioClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "minio error"})
		return
	}
	movie := Movie{Title: title, Description: desc}
	h.DB.Create(&movie)
	playlistPath, err := TranscodeAndUploadHLS(minioClient.Client, minioClient.Bucket, file, fileHeader, movie.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "ffmpeg or upload error", "details": err.Error()})
		return
	}
	movie.ObjectName = playlistPath
	h.DB.Save(&movie)
	c.JSON(201, movie)
}

func TranscodeAndUploadHLS(minioClient *minio.Client, bucket string, srcFile multipart.File, srcHeader *multipart.FileHeader, movieID uint) (string, error) {
	tmpDir, err := ioutil.TempDir("", "hls")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, srcHeader.Filename)
	outDir := filepath.Join(tmpDir, "hls")
	os.Mkdir(outDir, 0755)
	outPlaylist := filepath.Join(outDir, "playlist.m3u8")

	outFile, err := os.Create(inputPath)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(outFile, srcFile); err != nil {
		return "", err
	}
	outFile.Close()

	cmd := exec.Command(
		"ffmpeg", "-i", inputPath,
		"-c", "copy", "-start_number", "0",
		"-hls_time", "10", "-hls_list_size", "0", "-f", "hls", outPlaylist,
	)
	if err := cmd.Run(); err != nil {
		return "", err
	}

	files, err := ioutil.ReadDir(outDir)
	if err != nil {
		return "", err
	}
	prefix := fmt.Sprintf("movie-%d/", movieID)
	for _, f := range files {
		localPath := filepath.Join(outDir, f.Name())
		objectName := prefix + f.Name()
		_, err := minioClient.FPutObject(
			context.Background(), bucket, objectName, localPath, minio.PutObjectOptions{},
		)
		if err != nil {
			return "", err
		}
	}
	return prefix + "playlist.m3u8", nil
}

func (h *Handlers) GetMovieStream(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var movie Movie
	if err := h.DB.First(&movie, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	minioClient, err := NewMinioClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "minio error"})
		return
	}
	url, err := minioClient.GetPresignedURL(movie.ObjectName, 2*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "presign error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (h *Handlers) ProxyHLS(c *gin.Context) {
	id := c.Param("id")
	segment := c.Param("segment")
	// Находим фильм по id
	var movie Movie
	if err := h.DB.First(&movie, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "movie not found"})
		return
	}
	if movie.ObjectName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no HLS for this movie"})
		return
	}
	// Определяем путь к объекту в MinIO
	prefix := movie.ObjectName[:len(movie.ObjectName)-len("playlist.m3u8")]
	objectName := prefix + segment

	minioClient, err := NewMinioClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "minio error"})
		return
	}
	obj, err := minioClient.Client.GetObject(c, minioClient.Bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "minio get error"})
		return
	}
	stat, err := obj.Stat()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "segment not found"})
		return
	}
	// Определяем Content-Type
	if strings.HasSuffix(segment, ".m3u8") {
		c.Header("Content-Type", "application/vnd.apple.mpegurl")
	} else if strings.HasSuffix(segment, ".ts") {
		c.Header("Content-Type", "video/mp2t")
	} else {
		c.Header("Content-Type", stat.ContentType)
	}
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size))
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, obj)
}

func (h *Handlers) DeleteMovie(c *gin.Context) {
	roles, _ := c.Get("user_roles")
	isAdmin := false
	for _, r := range roles.([]string) {
		if r == "admin" {
			isAdmin = true
		}
	}
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
		return
	}
	id := c.Param("id")
	var movie Movie
	if err := h.DB.First(&movie, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	// Удаляем все файлы из MinIO
	if movie.ObjectName != "" {
		minioClient, err := NewMinioClient()
		if err == nil {
			prefix := movie.ObjectName[:len(movie.ObjectName)-len("playlist.m3u8")]
			ctx := context.Background()
			// Получаем список файлов с этим префиксом
			objectsCh := minioClient.Client.ListObjects(ctx, minioClient.Bucket, minio.ListObjectsOptions{
				Prefix:    prefix,
				Recursive: true,
			})
			for obj := range objectsCh {
				if obj.Err == nil {
					_ = minioClient.Client.RemoveObject(ctx, minioClient.Bucket, obj.Key, minio.RemoveObjectOptions{})
				}
			}
		}
	}
	// Удаляем фильм из базы
	h.DB.Delete(&movie)
	c.JSON(http.StatusOK, gin.H{"result": "deleted"})
}
