package go_client

import (
	"os"
	"fmt"
	//"log"
	"bytes"
	"net/url"
	"strings"
	"../httpgen_common"
)

type GoGenerator struct {
	Options *httpgen_common.CurlOptions
	Modules map[string]bool

	Data string
	DataVariable string
	ContentType string

	extraUrl string
}

func NewGoGenerator(options *httpgen_common.CurlOptions) *GoGenerator {
	result := &GoGenerator{Options: options}
	result.Modules = make(map[string]bool)
	result.Modules["net/http"] = true
	result.Modules["log"] = true
	result.Modules["io/ioutil"] = true
	return result
}

//--- Getter from template

func (self GoGenerator) Url() string {
	return fmt.Sprintf("\"%s\"%s", self.Options.Url, self.extraUrl)
}

func (self GoGenerator) Method() string {
	return self.Options.Method()
}

func (self GoGenerator) FilePath() string {
	return ""
}

//--- Setter methods

func (self *GoGenerator) SetDataForBody() {
	var buffer bytes.Buffer
	if len(self.Options.ProcessedData) == 1 {
		var body string
		body, self.DataVariable = NewStringForData(self, &self.Options.ProcessedData[0])
		buffer.WriteString(body)
	} else {
		for i, data := range self.Options.ProcessedData {
			if i > 0 {
				buffer.WriteString("    buffer.WriteByte('&')\n")
			} else {
				buffer.WriteString("var buffer bytes.Buffer\n")
			}
			buffer.WriteString(StringForData(self, &data))
		}
		self.DataVariable = "&buffer"
	}
	self.Data = buffer.String()
}

func (self *GoGenerator) SetDataForUrl() {
	if canUseSimpleForm(&self.Options.ProcessedData) {
		// Use url.Values to create URL option string
		self.SetDataForForm()
		self.extraUrl = " + \"?\" + values.Encode()"
	} else {
		// Use bytes.Buffer to create URL option string
		self.SetDataForBody()
		self.extraUrl = " + \"?\" + buffer.String()"
	}
}

func (self *GoGenerator) SetFormForBody() {
	var buffer bytes.Buffer
	buffer.WriteString("var buffer bytes.Buffer\n")
	buffer.WriteString("    writer := multipart.NewWriter(&buffer)\n")
	for _, data := range self.Options.ProcessedData {
		buffer.WriteString(FormString(self, &data))
	}
	self.Data = buffer.String()
}

//--- Pre code generators

func (self *GoGenerator) SetDataForForm() {
	entries := make(map[string][]string)
	for _, data := range self.Options.ProcessedData {
		singleData, _ := url.ParseQuery(data.Value)
		for key, values := range singleData {
			entry, ok := entries[key]
			if !ok {
				entry = make([]string, 0)
			}
			entries[key] = append(entry, values[0])
		}
	}

	var buffer bytes.Buffer
	count := 0
	for key, values := range entries {
		if count == 0 {
			buffer.WriteString("values := url.Values{\n")
		} else {
			buffer.WriteString(", \"")
		}
		buffer.WriteString("        \"" + key)
		buffer.WriteString("\": {")
		for j, value := range values {
			if j == 0 {
				buffer.WriteString("\"")
			} else {
				buffer.WriteString(", \"")
			}
			buffer.WriteString(value)
			buffer.WriteString("\"")
		}
		count++
		buffer.WriteString("},\n")
	}
	buffer.WriteString("    }\n")

	self.Data = buffer.String()
	self.DataVariable = "values"
	self.Modules["net/url"] = true
}

func NewStringForData(generator *GoGenerator, data *httpgen_common.DataOption) (string, string) {
	var result string
	var name string
	switch data.Type {
	case httpgen_common.DataAsciiType:
		if strings.HasPrefix(data.Value, "@") {
			var buffer bytes.Buffer
			buffer.WriteString("var buffer bytes.Buffer\n")
			buffer.WriteString("    content, err := ioutil.ReadFile(\"")
			buffer.WriteString(data.Value[1:])
			buffer.WriteString("\")\n")
			buffer.WriteString("    if err != nil {\n")
			buffer.WriteString("        log.Fatal(err)\n")
			buffer.WriteString("    }\n")
			buffer.WriteString("    buffer.WriteString(strings.Replace(string(content), \"\\n\", \"\", -1))")
			result = buffer.String()
			name = "&buffer"
			generator.Modules["strings"] = true
		} else {
			result = fmt.Sprintf("buffer := bytes.NewBufferString(\"%s\")\n", escapeDQ(strings.Replace(data.Value, "\n", "", -1)))
			name = "buffer"
		}
		generator.Modules["bytes"] = true
	case httpgen_common.DataBinaryType:
		if strings.HasPrefix(data.Value, "@") {
			var buffer bytes.Buffer
			buffer.WriteString("file, err := os.Open(\"")
			buffer.WriteString(data.Value[1:])
			buffer.WriteString("\")\n")
			buffer.WriteString("    if err != nil {\n")
			buffer.WriteString("        log.Fatal(err)\n")
			buffer.WriteString("    }\n")
			result = buffer.String()
			name = "file"
			generator.Modules["os"] = true
		} else {
			result = fmt.Sprintf("buffer := bytes.NewBufferString(\"%s\")\n", escapeDQ(data.Value))
			name = "buffer"
			generator.Modules["bytes"] = true
		}
	case httpgen_common.DataUrlEncodeType:
		if strings.HasPrefix(data.Value, "@") {
			var buffer bytes.Buffer
			buffer.WriteString("var buffer bytes.Buffer\n")
			buffer.WriteString("    content, err := ioutil.ReadFile(\"")
			buffer.WriteString(data.Value[1:])
			buffer.WriteString("\")\n")
			buffer.WriteString("    if err != nil {\n")
			buffer.WriteString("        log.Fatal(err)\n")
			buffer.WriteString("    }\n")
			buffer.WriteString("    buffer.WriteString(url.QueryEscape(string(content)))")
			result = buffer.String()
			name = "&buffer"
		} else {
			result = fmt.Sprintf("buffer := bytes.NewBufferString(url.QueryEscape(\"%s\"))\n", escapeDQ(data.Value))
			name = "buffer"
		}
		generator.Modules["bytes"] = true
		generator.Modules["net/url"] = true
	default:
		panic(fmt.Sprintf("unknown type: %d", data.Type))
	}
	return result, name
}

func StringForData(generator *GoGenerator, data *httpgen_common.DataOption) string {
	var result string
	switch data.Type {
	case httpgen_common.DataAsciiType:
		if strings.HasPrefix(data.Value, "@") {
			var buffer bytes.Buffer
			buffer.WriteString("    {\n")
			buffer.WriteString(fmt.Sprintf("        content, err := ioutil.ReadFile(\"%s\")\n", data.Value[1:]))
			buffer.WriteString("        if err != nil {\n")
			buffer.WriteString("            log.Fatal(err)\n")
			buffer.WriteString("        }\n")
			buffer.WriteString("        buffer.WriteString(strings.Replace(string(content), \"\\n\", \"\", -1))\n")
			buffer.WriteString("    }\n")
			result = buffer.String()
			generator.Modules["strings"] = true
		} else {
			result = fmt.Sprintf("    buffer.WriteString(\"%s\")\n", escapeDQ(strings.Replace(data.Value, "\n", "", -1)))
		}
	case httpgen_common.DataBinaryType:
		if strings.HasPrefix(data.Value, "@") {
			var buffer bytes.Buffer
			buffer.WriteString("    {\n")
			buffer.WriteString(fmt.Sprintf("        file, err := os.Open(\"%s\")\n", data.Value[1:]))
			buffer.WriteString("        if err != nil {\n")
			buffer.WriteString("            log.Fatal(err)\n")
			buffer.WriteString("        }\n")
			buffer.WriteString("        io.Copy(&buffer, file)\n")
			buffer.WriteString("    }\n")
			result = buffer.String()
			generator.Modules["os"] = true
			generator.Modules["io"] = true
		} else {
			result = fmt.Sprintf("    buffer.WriteString(\"%s\")\n", escapeDQ(data.Value))
		}
	case httpgen_common.DataUrlEncodeType:
		if strings.HasPrefix(data.Value, "@") {
			var buffer bytes.Buffer
			buffer.WriteString("    {\n")
			buffer.WriteString(fmt.Sprintf("        content, err := ioutil.ReadFile(\"%s\")\n", data.Value[1:]))
			buffer.WriteString("        if err != nil {\n")
			buffer.WriteString("            log.Fatal(err)\n")
			buffer.WriteString("        }\n")
			buffer.WriteString("        buffer.WriteString(url.QueryEscape(string(content)))\n")
			buffer.WriteString("    }\n")
			result = buffer.String()
		} else {
			result = fmt.Sprintf("    buffer.WriteString(url.QueryEscape(\"%s\"))\n", escapeDQ(data.Value))
		}
		generator.Modules["net/url"] = true
	default:
		panic(fmt.Sprintf("unknown type: %d", data.Type))
	}
	generator.Modules["bytes"] = true
	return result
}

func FormString(generator *GoGenerator, data *httpgen_common.DataOption) string {
	var result string
	switch data.Type {
	case httpgen_common.FormType:
		field := strings.SplitN(data.Value, "=", 2)
		if len(field) != 2 {
			fmt.Fprintln(os.Stderr, "Warning: Illegally formatted input field!\ncurl: option -F: is badly used here")
			os.Exit(1)
		}
		if strings.HasPrefix(field[1], "@") {
			var buffer bytes.Buffer
			var contentType string
			fragments := strings.Split(field[1][1:], ";")
			sourceFile := fragments[0]
			sentFileName := fragments[0]
			for _, fragment := range fragments[1:] {
				if strings.HasPrefix(fragment, "filename=") {
					sentFileName = fragment[9:]
				} else if strings.HasPrefix(fragment, "type=") {
					contentType = fragment[5:]
				}
			}
			buffer.WriteString("    {\n")
			if contentType != "" {
				buffer.WriteString("        header := make(textproto.MIMEHeader)\n")
				buffer.WriteString(fmt.Sprintf("        header.Add(\"Content-Disposition\", \"form-data; name=\\\"%s\\\"; filename=\\\"%s\\\"\")\n", field[0], sentFileName))
				buffer.WriteString(fmt.Sprintf("        header.Add(\"Content-Type\", \"%s\")\n", contentType))
				buffer.WriteString("        fileWriter, err := writer.CreatePart(header)\n")
				buffer.WriteString("        if err != nil {\n")
				buffer.WriteString("            log.Fatal(err)\n")
				buffer.WriteString("        }\n")
				generator.Modules["net/textproto"] = true
			} else {
				buffer.WriteString(fmt.Sprintf("        fileWriter, err := writer.CreateFormFile(\"%s\", \"%s\")\n", field[0], sentFileName))
				buffer.WriteString("        if err != nil {\n")
				buffer.WriteString("            log.Fatal(err)\n")
				buffer.WriteString("        }\n")
			}
			buffer.WriteString(fmt.Sprintf("        file, err := os.Open(\"%s\")\n", sourceFile))
			buffer.WriteString("        if err != nil {\n")
			buffer.WriteString("            log.Fatal(err)\n")
			buffer.WriteString("        }\n")
			buffer.WriteString("        io.Copy(fileWriter, file)\n")
			buffer.WriteString("    }\n")
			result = buffer.String()
			generator.Modules["os"] = true
			generator.Modules["io"] = true
		} else if strings.HasPrefix(field[1], "<") {
			var buffer bytes.Buffer
			var contentType string
			fragments := strings.Split(field[1][1:], ";")
			sourceFile := fragments[0]
			for _, fragment := range fragments[1:] {
				if strings.HasPrefix(fragment, "type=") {
					contentType = fragment[5:]
				}
			}
			buffer.WriteString("    {\n")
			buffer.WriteString("        header := make(textproto.MIMEHeader)\n")
			buffer.WriteString(fmt.Sprintf("        header.Add(\"Content-Disposition\", \"form-data; name=\\\"%s\\\"\")\n", field[0]))
			if contentType != "" {
				buffer.WriteString(fmt.Sprintf("        header.Add(\"Content-Type\", \"%s\")\n", contentType))
			}
			buffer.WriteString("        fileWriter, err := writer.CreatePart(header)\n")
			buffer.WriteString("        if err != nil {\n")
			buffer.WriteString("            log.Fatal(err)\n")
			buffer.WriteString("        }\n")
			buffer.WriteString(fmt.Sprintf("        file, err := os.Open(\"%s\")\n", sourceFile))
			buffer.WriteString("        if err != nil {\n")
			buffer.WriteString("            log.Fatal(err)\n")
			buffer.WriteString("        }\n")
			buffer.WriteString("        io.Copy(fileWriter, file)\n")
			buffer.WriteString("    }\n")
			result = buffer.String()
			generator.Modules["net/textproto"] = true
			generator.Modules["os"] = true
			generator.Modules["io"] = true
		} else {
			result = fmt.Sprintf("    writer.WriteField(\"%s\", \"%s\")\n", field[0], field[1])
		}
	case httpgen_common.FormStringType:
		field := strings.SplitN(data.Value, "=", 2)
		if len(field) != 2 {
			fmt.Fprintln(os.Stderr, "Warning: Illegally formatted input field!\ncurl: option -F: is badly used here")
			os.Exit(1)
		}
		result = fmt.Sprintf("    writer.WriteField(\"%s\", \"%s\")\n", field[0], field[1])
	}
	generator.Modules["bytes"] = true
	generator.Modules["mime/multipart"] = true
	return result
}