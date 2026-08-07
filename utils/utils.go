package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
	"gorm.io/gorm"
)

const (
	ContentTypeJSON           = "application/json"
	ContentTypeFormURLEncoded = "application/x-www-form-urlencoded"
)

// ParseRequestBody parses the request body based on the Content-Type header
// It handles both JSON and form-urlencoded data, and allows for field validation.
func ParseRequestBody(r *http.Request, v interface{}, allowedFields []string) error {
	contentType := r.Header.Get("Content-Type")

	// Handle JSON request
	if strings.HasPrefix(contentType, ContentTypeJSON) {
		var tempMap map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&tempMap); err != nil {
			return errors.New(MsgInvalidJSONData)
		}
		// Validate fields
		if err := ValidateFields(tempMap, allowedFields); err != nil {
			return err
		}

		// Convert map to the struct
		bytes, _ := json.Marshal(tempMap)
		if err := json.Unmarshal(bytes, v); err != nil {
			return errors.New(MsgInvalidJSONData)
		}
		return nil
	}

	// Handle x-www-form-urlencoded request
	if strings.HasPrefix(contentType, ContentTypeFormURLEncoded) {
		if err := r.ParseForm(); err != nil {
			return errors.New(MsgInvalidFormData)
		}

		// Map form values and validate fields
		formValues := r.PostForm
		tempMap := make(map[string]interface{})
		for key, values := range formValues {
			if len(values) > 0 {
				tempMap[key] = values[0]
			}
		}

		if err := ValidateFields(tempMap, allowedFields); err != nil {
			return err
		}

		// Convert form values to struct
		err := mapFormToStruct(tempMap, v)
		if err != nil {
			return err
		}
		return nil
	}

	return errors.New(MsgUnsupportedContentType)
}

// ValidateFields ensures only allowed fields are present in the request body
func ValidateFields(data map[string]interface{}, allowedFields []string) error {
	allowedFieldsMap := make(map[string]bool)
	for _, field := range allowedFields {
		allowedFieldsMap[field] = true
	}

	for field := range data {
		if !allowedFieldsMap[field] {
			return errors.New(MsgOnlyAllowedFieldsAllowed)
		}
	}

	return nil
}

// AllowFields checks if the provided fields in the query params are within the allowed fields list.
func AllowFields(query url.Values, allowedFields []string) bool {
	// Create a map for allowed fields for quick lookup
	allowedMap := make(map[string]bool)
	for _, field := range allowedFields {
		allowedMap[field] = true
	}

	// If no query params are passed, allow the request
	if len(query) == 0 {
		return true
	}

	// Iterate over the query params and ensure that all fields are in the allowed list
	for param := range query {
		if !allowedMap[param] {
			return false
		}
	}

	return true
}

// mapFormToStruct maps form values to a struct
func mapFormToStruct(data map[string]interface{}, v interface{}) error {
	val := reflect.ValueOf(v).Elem()
	typ := val.Type()

	for key, value := range data {
		// Normalize the form key and struct field name to lowercase for case-insensitive matching
		normalizedKey := strings.ToLower(key)

		for i := 0; i < val.NumField(); i++ {
			structField := typ.Field(i)

			// Use form tag if available
			formTag := structField.Tag.Get("form")
			if formTag != "" && formTag == normalizedKey {
				field := val.FieldByName(structField.Name)
				if field.IsValid() && field.CanSet() {
					fieldValue := reflect.ValueOf(value)
					if fieldValue.Type().AssignableTo(field.Type()) {
						field.Set(fieldValue)
					}
				}
				break
			}
		}
	}
	return nil
}

// GenerateUniqueString generates a unique string for filenames
func GenerateUniqueString() string {
	return fmt.Sprintf("%d_%d", time.Now().UnixNano(), rand.Intn(1000))
}

// TimePtr returns a pointer to the given time.Time value
func TimePtr(t time.Time) *time.Time {
	return &t
}

// GetFileExtension returns the file extension (with dot) from a filename, e.g., '.png', '.jpg'.
func GetFileExtension(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx == -1 {
		return ""
	}
	return filename[idx:]
}

// ComputeFileHashFromFormFile computes the SHA256 hash of a gin FormFile and returns the hash and file bytes
func ComputeFileHashFromFormFile(file *multipart.FileHeader) (string, []byte, error) {
	f, err := file.Open()
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	// Read all bytes
	fileBytes, err := io.ReadAll(f)
	if err != nil {
		return "", nil, err
	}
	// Compute hash
	h := sha256.New()
	h.Write(fileBytes)
	hash := hex.EncodeToString(h.Sum(nil))
	return hash, fileBytes, nil
}

// DecryptAndReEncryptField decrypts an encrypted field and then re-encrypts it for DB storage.
func DecryptAndReEncryptField(encryptedValue string, fieldName string) (string, string) {
	decrypted, err := Decrypt(encryptedValue)
	if err != nil {
		return "", "Invalid encrypted " + fieldName
	}
	reEncrypted, err := Encrypt(decrypted)
	if err != nil {
		return "", "Failed to encrypt " + fieldName + " for DB"
	}
	return reEncrypted, ""
}

// DecryptAndReEncryptFields decrypts and re-encrypts multiple fields, rolling back the transaction if any fail.
func DecryptAndReEncryptFields(tx *gorm.DB, fields map[string]string) (map[string]string, string) {
	result := make(map[string]string)
	for key, val := range fields {
		if val == "" {
			continue
		}
		reEncrypted, errMsg := DecryptAndReEncryptField(val, key)
		if errMsg != "" {
			tx.Rollback()
			return nil, errMsg
		}
		result[key] = reEncrypted
	}
	return result, ""
}
