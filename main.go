package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// Simple XML structure to check for model declaration
type Model struct {
	XMLName xml.Name `xml:"model"`
	Name    string   `xml:"name,attr"`
}

// Templates for generated files
const modulePropertiesTmpl = `module.id={{.Name}}
module.title={{.Name}}
module.description={{.Name}}
module.version={{.Version}}
`

const moduleContextXmlTmpl = `<?xml version='1.0' encoding='UTF-8'?>
<beans xmlns="http://www.springframework.org/schema/beans"
       xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
       xsi:schemaLocation="http://www.springframework.org/schema/beans
          http://www.springframework.org/schema/beans/spring-beans-3.0.xsd">
    <bean id="{{.Name}}" parent="dictionaryModelBootstrap" depends-on="dictionaryBootstrap">
        <property name="models">
            <list>
                {{- range .ModelPaths}}
                <value>{{.}}</value>
                {{- end}}
            </list>
        </property>
    </bean>
</beans>`

type ModuleData struct {
	Name       string
	Version    string
	ModelPaths []string
}

// Function to extract and parse module.properties from ZIP
func getModuleVersion(zipReader *zip.ReadCloser, moduleName string) (string, error) {
	propertiesPath := fmt.Sprintf("alfresco/module/%s/module.properties", moduleName)
	for _, file := range zipReader.File {
		if file.Name == propertiesPath {
			rc, err := file.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			scanner := bufio.NewScanner(rc)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "module.version=") {
					return strings.TrimPrefix(line, "module.version="), nil
				}
			}
			return "", scanner.Err()
		}
	}
	return "1.0.0", nil // Default version if not found
}

// Function to increment version
func incrementVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		// If version is incomplete, pad with zeros
		for len(parts) < 3 {
			parts = append(parts, "0")
		}
	}

	// Try to increment the last number
	if lastNum, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
		parts[len(parts)-1] = strconv.Itoa(lastNum + 1)
	} else {
		// If parsing fails, append .1
		parts = append(parts, "1")
	}

	return strings.Join(parts, ".")
}

func main() {
	// Parse command line arguments
	zipFile := flag.String("zip", "", "Path to ZIP file to process")
	outputJar := flag.String("output", "models.jar", "Output JAR file name")
	flag.Parse()

	if *zipFile == "" {
		log.Fatal("Please provide a ZIP file path using -zip flag")
	}

	// Get module name from ZIP filename, removing version information
	moduleName := cleanModuleName(*zipFile)

	// Open the ZIP file
	reader, err := zip.OpenReader(*zipFile)
	if err != nil {
		log.Fatalf("Failed to open ZIP file: %v", err)
	}
	defer reader.Close()

	// Get current version from module.properties
	currentVersion, err := getModuleVersion(reader, moduleName)
	if err != nil {
		log.Printf("Warning: Could not read current version: %v", err)
		currentVersion = "1.0.0"
	}

	// Increment the version
	newVersion := incrementVersion(currentVersion)

	// Create temporary directory for XML files
	tempDir, err := os.MkdirTemp("", "alfresco-models")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Process ZIP contents
	modelFiles := make([]string, 0)
	for _, file := range reader.File {
		if strings.HasSuffix(strings.ToLower(file.Name), ".xml") {
			if isAlfrescoModel(file) {
				// Copy file to temp directory
				destPath := filepath.Join(tempDir, filepath.Base(file.Name))
				if err := extractFile(file, destPath); err != nil {
					log.Printf("Failed to extract %s: %v", file.Name, err)
					continue
				}
				modelFiles = append(modelFiles, destPath)
			}
		}
	}

	if len(modelFiles) == 0 {
		log.Fatal("No Alfresco content model XML files found")
	}

	// Create JAR file with module structure and new version
	if err := createModuleJar(*outputJar, modelFiles, moduleName, newVersion); err != nil {
		log.Fatalf("Failed to create JAR file: %v", err)
	}

	fmt.Printf("Successfully created JAR file %s with %d model files (version %s)\n", 
		*outputJar, len(modelFiles), newVersion)
}

func cleanModuleName(filename string) string {
	// Remove file extension
	name := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))

	// Regular expression to match version patterns:
	// - Matches patterns like "-1.0.0", "-1.0", "-v1.0.0", "_1.0.0", "_v1.0.0"
	// - Handles both hyphen and underscore separators
	// - Handles optional 'v' prefix before version number
	versionRegex := regexp.MustCompile(`[-_]v?\d+(\.\d+)*(-SNAPSHOT)?$`)

	// Remove version information
	cleanName := versionRegex.ReplaceAllString(name, "")

	return cleanName
}

func isAlfrescoModel(file *zip.File) bool {
	rc, err := file.Open()
	if err != nil {
		return false
	}
	defer rc.Close()

	// Read the first few KB to check for model declaration
	buffer := make([]byte, 4096)
	n, err := rc.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	// Check if it contains model declaration
	content := string(buffer[:n])
	return strings.Contains(content, "<model") && strings.Contains(content, "name=")
}

func extractFile(file *zip.File, destPath string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, rc)
	return err
}

// Helper function to create a directory entry in the ZIP
func createDirInZip(zipWriter *zip.Writer, name string) error {
	if !strings.HasSuffix(name, "/") {
		name = name + "/"
	}
	header := &zip.FileHeader{
		Name:     name,
		Method:   zip.Store, // Directories should use STORE method
		Modified: time.Now(),
	}
	header.SetMode(0755 | os.ModeDir)
	_, err := zipWriter.CreateHeader(header)
	return err
}

// Helper function to create a file in the ZIP with current timestamp
func createFileInZip(zipWriter *zip.Writer, name string, compress bool) (io.Writer, error) {
	header := &zip.FileHeader{
		Name:     name,
		Modified: time.Now(),
	}
	if compress {
		header.Method = zip.Deflate
	} else {
		header.Method = zip.Store
	}
	header.SetMode(0644)
	return zipWriter.CreateHeader(header)
}

func createModuleJar(jarPath string, files []string, moduleName, version string) error {
	jarFile, err := os.Create(jarPath)
	if err != nil {
		return err
	}
	defer jarFile.Close()

	zipWriter := zip.NewWriter(jarFile)
	defer zipWriter.Close()

	// Create all necessary directories first
	directories := []string{
		"META-INF/",
		fmt.Sprintf("alfresco/"),
		fmt.Sprintf("alfresco/module/"),
		fmt.Sprintf("alfresco/module/%s/", moduleName),
		fmt.Sprintf("alfresco/module/%s/model/", moduleName),
	}

	// Sort directories to ensure parent directories are created first
	sort.Strings(directories)
	for _, dir := range directories {
		if err := createDirInZip(zipWriter, dir); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	// Create META-INF/MANIFEST.MF
	manifest := []byte(fmt.Sprintf("Manifest-Version: 1.0\n"+
		"Created-By: Alfresco Model Extractor\n"+
		"Built-By: %s\n"+
		"Build-Jdk: 17.0.5\n"+
		"Package: org.alfresco.module\n"+
		"Implementation-Version: %s\n"+
		"Implementation-Title: %s\n\n",
		os.Getenv("USER"),
		version,
		moduleName))

	manifestWriter, err := createFileInZip(zipWriter, "META-INF/MANIFEST.MF", false)
	if err != nil {
		return err
	}
	if _, err := manifestWriter.Write(manifest); err != nil {
		return err
	}

	// Prepare model paths for module-context.xml
	var modelPaths []string
	for _, file := range files {
		modelPath := fmt.Sprintf("alfresco/module/%s/model/%s", moduleName, filepath.Base(file))
		// Ensure forward slashes
		modelPath = strings.ReplaceAll(modelPath, "\\", "/")
		modelPaths = append(modelPaths, modelPath)
	}

	// Sort model paths for consistency
	sort.Strings(modelPaths)

	// Prepare module data for templates with version
	moduleData := ModuleData{
		Name:       moduleName,
		Version:    version,
		ModelPaths: modelPaths,
	}

	// Create module.properties
	propsTemplate := template.Must(template.New("properties").Parse(modulePropertiesTmpl))
	var propsBuffer bytes.Buffer
	if err := propsTemplate.Execute(&propsBuffer, moduleData); err != nil {
		return err
	}
	propsWriter, err := createFileInZip(zipWriter, fmt.Sprintf("alfresco/module/%s/module.properties", moduleName), true)
	if err != nil {
		return err
	}
	if _, err := propsWriter.Write(propsBuffer.Bytes()); err != nil {
		return err
	}

	// Create module-context.xml
	contextTemplate := template.Must(template.New("context").Parse(moduleContextXmlTmpl))
	var contextBuffer bytes.Buffer
	if err := contextTemplate.Execute(&contextBuffer, moduleData); err != nil {
		return err
	}
	contextWriter, err := createFileInZip(zipWriter, fmt.Sprintf("alfresco/module/%s/module-context.xml", moduleName), true)
	if err != nil {
		return err
	}
	if _, err := contextWriter.Write(contextBuffer.Bytes()); err != nil {
		return err
	}

	// Add XML files to JAR in the module's model directory
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		fileName := fmt.Sprintf("alfresco/module/%s/model/%s", moduleName, filepath.Base(file))
		// Ensure forward slashes
		fileName = strings.ReplaceAll(fileName, "\\", "/")

		writer, err := createFileInZip(zipWriter, fileName, true)
		if err != nil {
			return err
		}

		if _, err := writer.Write(content); err != nil {
			return err
		}
	}

	return nil
}
