package utils

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg" // Registra el decodificador JPEG
	_ "image/png"  // Registra el decodificador PNG
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/chai2010/webp"
	"github.com/nfnt/resize"
)

type ConfigOSS struct {
	Productos ConfigProductosOSS `yaml:"productos"`
}

type ConfigProductosOSS struct {
	OssEndpoint        string `yaml:"oss_endpoint"`
	OssAccessKeyID     string `yaml:"oss_access_key_id"`
	OssAccessKeySecret string `yaml:"oss_access_key_secret"`
	OssBucketName      string `yaml:"oss_bucket_name"`
}

func NewConfigOSS(configPath string) (*ConfigOSS, error) {
	// 1. Leer el archivo YAML
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error leyendo config: %w", err)
	}

	var cfg ConfigOSS
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parseando yaml: %w", err)
	}

	return &cfg, nil
}

// ProcesarYSubirImagen optimiza la imagen a WebP, la redimensiona y la sube a Alibaba OSS
func (cfg *ConfigOSS) ProcesarYSubirImagen(fileHeader *multipart.FileHeader) (string, error) {
	// 1. Abrir el archivo subido
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 2. Decodificar la imagen (soporta JPG, PNG)
	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("error al decodificar imagen: %v", err)
	}

	// 3. Redimensionar (Ancho máximo 800px, manteniendo ratio de aspecto)
	// Si la imagen es más pequeña, se puede dejar igual, pero resize.Resize lo maneja bien.
	imgOptimizada := resize.Resize(800, 0, img, resize.Lanczos3)

	// 4. Codificar a WebP en un buffer de memoria
	var buf bytes.Buffer
	// Calidad 80 es el punto dulce entre peso y fidelidad visual
	err = webp.Encode(&buf, imgOptimizada, &webp.Options{Lossless: false, Quality: 80})
	if err != nil {
		return "", fmt.Errorf("error al convertir a WebP: %v", err)
	}

	// 5. Conectar con Alibaba Cloud OSS
	client, err := oss.New(cfg.Productos.OssEndpoint, cfg.Productos.OssAccessKeyID, cfg.Productos.OssAccessKeySecret)
	if err != nil {
		return "", fmt.Errorf("error conectando a OSS: %v", err)
	}

	bucket, err := client.Bucket(cfg.Productos.OssBucketName)
	if err != nil {
		return "", fmt.Errorf("error obteniendo el bucket: %v", err)
	}

	// 6. Generar nombre único con extensión .webp
	cleanName := strings.TrimSuffix(filepath.Base(fileHeader.Filename), filepath.Ext(fileHeader.Filename))
	objectKey := fmt.Sprintf("productos/fotos/%d_%s.webp", time.Now().UnixNano(), cleanName)

	// 7. Subir el buffer directamente a OSS
	err = bucket.PutObject(objectKey, &buf)
	if err != nil {
		return "", fmt.Errorf("error al subir a OSS: %v", err)
	}

	// 8. Retornar la URL pública del archivo
	url := fmt.Sprintf("https://%s.%s/%s", cfg.Productos.OssBucketName, cfg.Productos.OssEndpoint, objectKey)
	return url, nil
}

// SubirFichaTecnica sube archivos PDF/Docs sin modificarlos
func (cfg *ConfigOSS) SubirFichaTecnica(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	client, err := oss.New(cfg.Productos.OssEndpoint, cfg.Productos.OssAccessKeyID, cfg.Productos.OssAccessKeySecret)
	if err != nil {
		return "", err
	}

	bucket, err := client.Bucket(cfg.Productos.OssBucketName)
	if err != nil {
		return "", err
	}

	objectKey := fmt.Sprintf("productos/fichas/%d_%s", time.Now().Unix(), filepath.Base(fileHeader.Filename))

	err = bucket.PutObject(objectKey, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s.%s/%s", cfg.Productos.OssBucketName, cfg.Productos.OssEndpoint, objectKey), nil
}

// SubirEvidencia sube un archivo de evidencia (imagen o documento) tal cual a OSS
func (cfg *ConfigOSS) SubirEvidencia(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	client, err := oss.New(cfg.Productos.OssEndpoint, cfg.Productos.OssAccessKeyID, cfg.Productos.OssAccessKeySecret)
	if err != nil {
		return "", err
	}

	bucket, err := client.Bucket(cfg.Productos.OssBucketName)
	if err != nil {
		return "", err
	}

	objectKey := fmt.Sprintf("devoluciones/evidencias/%d_%s", time.Now().UnixNano(), filepath.Base(fileHeader.Filename))

	err = bucket.PutObject(objectKey, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s.%s/%s", cfg.Productos.OssBucketName, cfg.Productos.OssEndpoint, objectKey), nil
}
