# Alfresco Model Extractor

This program is a Go-based tool for processing and packaging Alfresco XML model files extracted from an existing Alfresco Addon into a JAR format. It extracts valid Alfresco content models from a ZIP archive, generates required module configuration files, and creates a JAR package with the desired structure.

Use this tool when upgrading an Alfresco installation that no longer supports a specific Alfresco Addon but still requires its Custom Content Model. Since existing content depends on this model, the Alfresco Model Extractor enables you to extract only the necessary Custom Content Model files from the original Addon and package them into a new Addon (JAR file) compatible with the upgraded Alfresco installation.

## Features

- **Extracts Alfresco Models**: Scans a JAR/AMP file containing an Alfresco Addon for Alfresco XML content models.
- **Modular Packaging**: Packages models into a JAR file for easy deployment in Alfresco.
- **Auto-Configuration**: Generates `module.properties` and `module-context.xml` files.

## Using

Download the appropriate binary for your OS and architecture from [Releases](https://github.com/aborroy/alfresco-model-extractor/releases):

- **macOS with Intel**: `alfresco-model-extractor_darwin_amd64`
- **macOS with Silicon**: `alfresco-model-extractor_darwin_arm64`
- **Linux**: `alfresco-model-extractor_linux_amd64`
- **Windows**: `alfresco-model-extractor_windows_amd64.exe`

> Each version is precompiled for the latest supported Go version.

### Make Executable (if necessary)

On macOS and Linux, you may need to make the binary executable:

```sh
chmod +x alfresco-model-extractor
```

### Command Line Arguments

- `-zip` (required): Path to the input Alfresco Addon file containing Alfresco models.
- `-output` (optional): Name of the output JAR file. Default is `models.jar`.

### Run with Command Line Options

Open a terminal (or Command Prompt on Windows) and navigate to the binary's folder. Run the program with the necessary arguments:

```sh
./alfresco-model-extractor -zip path/to/your-models.amp -output my-models.jar
```

Replace `path/to/your-models.amp` with the path to your Alfresco Addon file (AMP and JAR formats accepted) and specify your desired output JAR file name.

For instance, on macOS:

```sh
./alfresco-model-extractor_darwin_amd64 -zip original-addon.jar -output original-addon-models.jar
```

This will create `original-addon-models` containing only the Custom Content Model files needed for your updated Alfresco installation.

### Applying to a Real Use Case

[A namespace prefix is not registered - Simple OCR](https://connect.hyland.com/t5/alfresco-forum/a-namespace-prefix-is-not-registered-simple-ocr/td-p/483062) describes the scenario this program is designed for.

1. Download the [original repository addon](https://github.com/keensoft/alfresco-simple-ocr/releases/download/2.3.1/simple-ocr-repo-2.3.1.jar) that is not being deployed on the updated Alfresco installation from the original GitHub project: https://github.com/keensoft/alfresco-simple-ocr/releases/tag/2.3.1

2. Download the the appropriate Alfresco Model Extractor binary for your OS

3. Use the program to create the new Alfresco Addon as JAR

```sh
./alfresco-model-extractor -zip simple-ocr-repo-2.3.1.jar -output simple-ocr-repo-models-2.3.1.jar
```

4. Apply `simple-ocr-repo-models-2.3.1.jar` to the upgraded Alfresco deployment

## Using from source code

### Prerequisites

- Go 1.17+ installed on your system.

### Example

To use the program, run the following command:

```sh
go run main.go -zip path/to/your-models.zip -output my-models.jar
```

### Output

This will generate a JAR file with the following structure:

```sh
my-models.jar
└── alfresco/
    └── module/
        └── <module_name>/
            ├── module.properties
            ├── module-context.xml
            └── model/
                └── <your-model-files>.xml
```

## Installation

Clone the repository and install dependencies:

```sh
git clone https://github.com/yourusername/alfresco-model-extractor.git
cd alfresco-model-extractor
go mod download
```

## Build Instructions

To compile the program, run:

```sh
go build -o alfresco-model-extractor main.go
```

This will create an executable named `alfresco-model-extractor` in your directory.