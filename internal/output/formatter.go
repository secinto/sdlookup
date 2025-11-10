package output

import (
	"fmt"
	"io"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/h4sh5/sdlookup/internal/models"
)

// Formatter defines the interface for output formatters
type Formatter interface {
	Format(result *models.ScanResult) (string, error)
	FormatBatch(results []*models.ScanResult) (string, error)
}

// CSVFormatter formats output as CSV
type CSVFormatter struct {
	onlyIPPort bool
}

// NewCSVFormatter creates a new CSV formatter
func NewCSVFormatter(onlyIPPort bool) *CSVFormatter {
	return &CSVFormatter{
		onlyIPPort: onlyIPPort,
	}
}

// Format formats a single result as CSV
func (f *CSVFormatter) Format(result *models.ScanResult) (string, error) {
	if result.Error != nil {
		return "", result.Error
	}

	if result.Info == nil {
		return "", nil
	}

	// If no ports, show IP with metadata (for omitEmpty=false case)
	if len(result.Info.Ports) == 0 {
		if f.onlyIPPort {
			return fmt.Sprintf("%s", result.IP), nil
		}
		hostnames := strings.Join(result.Info.Hostnames, ";")
		tags := strings.Join(result.Info.Tags, ";")
		cpes := strings.Join(result.Info.Cpes, ";")
		vulns := strings.Join(result.Info.Vulns, ";")
		return fmt.Sprintf("%s,(no ports),%s,%s,%s,%s", result.IP, hostnames, tags, cpes, vulns), nil
	}

	var lines []string
	for _, port := range result.Info.Ports {
		line := fmt.Sprintf("%s:%d", result.IP, port)

		if !f.onlyIPPort {
			hostnames := strings.Join(result.Info.Hostnames, ";")
			tags := strings.Join(result.Info.Tags, ";")
			cpes := strings.Join(result.Info.Cpes, ";")
			vulns := strings.Join(result.Info.Vulns, ";")
			line = fmt.Sprintf("%s,%s,%s,%s,%s", line, hostnames, tags, cpes, vulns)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n"), nil
}

// FormatBatch formats multiple results as CSV
func (f *CSVFormatter) FormatBatch(results []*models.ScanResult) (string, error) {
	var lines []string
	for _, result := range results {
		line, err := f.Format(result)
		if err != nil {
			continue
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n"), nil
}

// JSONFormatter formats output as JSON
type JSONFormatter struct {
	pretty bool
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(pretty bool) *JSONFormatter {
	return &JSONFormatter{pretty: pretty}
}

// Format formats a single result as JSON
func (f *JSONFormatter) Format(result *models.ScanResult) (string, error) {
	if result.Error != nil {
		return "", result.Error
	}

	if result.Info == nil {
		return "", nil
	}

	var data []byte
	var err error

	if f.pretty {
		data, err = jsoniter.MarshalIndent(result.Info, "", "  ")
	} else {
		data, err = jsoniter.Marshal(result.Info)
	}

	if err != nil {
		return "", fmt.Errorf("marshaling JSON: %w", err)
	}

	return string(data), nil
}

// FormatBatch formats multiple results as JSON
func (f *JSONFormatter) FormatBatch(results []*models.ScanResult) (string, error) {
	var infos []*models.ShodanIPInfo
	for _, result := range results {
		if result.Error == nil && result.Info != nil {
			infos = append(infos, result.Info)
		}
	}

	var data []byte
	var err error

	if f.pretty {
		data, err = jsoniter.MarshalIndent(infos, "", "  ")
	} else {
		data, err = jsoniter.Marshal(infos)
	}

	if err != nil {
		return "", fmt.Errorf("marshaling JSON: %w", err)
	}

	return string(data), nil
}

// SimpleFormatter formats output as simple IP:Port
type SimpleFormatter struct{}

// NewSimpleFormatter creates a new simple formatter
func NewSimpleFormatter() *SimpleFormatter {
	return &SimpleFormatter{}
}

// Format formats a single result as simple IP:Port
func (f *SimpleFormatter) Format(result *models.ScanResult) (string, error) {
	if result.Error != nil {
		return "", result.Error
	}

	if result.Info == nil {
		return "", nil
	}

	// If no ports, just show IP (for omitEmpty=false case)
	if len(result.Info.Ports) == 0 {
		return result.IP, nil
	}

	var lines []string
	for _, port := range result.Info.Ports {
		lines = append(lines, fmt.Sprintf("%s:%d", result.IP, port))
	}

	return strings.Join(lines, "\n"), nil
}

// FormatBatch formats multiple results as simple IP:Port
func (f *SimpleFormatter) FormatBatch(results []*models.ScanResult) (string, error) {
	var lines []string
	for _, result := range results {
		line, err := f.Format(result)
		if err != nil {
			continue
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n"), nil
}

// Writer manages output writing
type Writer struct {
	formatter Formatter
	output    io.Writer
}

// NewWriter creates a new output writer
func NewWriter(formatter Formatter, output io.Writer) *Writer {
	return &Writer{
		formatter: formatter,
		output:    output,
	}
}

// Write writes a single result
func (w *Writer) Write(result *models.ScanResult) error {
	output, err := w.formatter.Format(result)
	if err != nil {
		return err
	}

	if output == "" {
		return nil
	}

	_, err = fmt.Fprintln(w.output, output)
	return err
}

// WriteBatch writes multiple results
func (w *Writer) WriteBatch(results []*models.ScanResult) error {
	output, err := w.formatter.FormatBatch(results)
	if err != nil {
		return err
	}

	if output == "" {
		return nil
	}

	_, err = fmt.Fprintln(w.output, output)
	return err
}
