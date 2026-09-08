package utl

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml/lexer"
	"github.com/goccy/go-yaml/token"
	"gopkg.in/yaml.v3"
)

// Convert byte slice to YAML interface object

// Convert YAML interface object to byte slice, with option indent spacing
func yamlToBytesIndent(yamlObject any, indent int) (yamlBytes []byte, err error) {
	buffer := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buffer)
	encoder.SetIndent(indent)
	err = encoder.Encode(yamlObject)
	if err != nil {
		return nil, err
	}
	yamlBytes = buffer.Bytes()
	return yamlBytes, nil
}

// With default 2 space indent
func yamlToBytes(yamlObject any) (yamlBytes []byte, err error) {
	indent := 2
	yamlBytes, err = yamlToBytesIndent(yamlObject, indent)
	return yamlBytes, err
}

// Convert YAML interface object to a colorized string representation

// Convert YAML object to bytes

// Create a pipe to capture the output

// Redirect standard output to the pipe

// Print the colorized YAML bytes

// Restore the standard output

// Read the output from the pipe

// Return the colorized string

// Print YAML object

// Colorize given token. Internal helper function.
func colorizeString(tk *token.Token, src string) string {
	str := Whi(src)
	switch tk.Type {
	case token.MappingKeyType:
		str = Blu(src)
	case token.StringType, token.SingleQuoteType, token.DoubleQuoteType:
		prev := tk.PreviousType()
		next := tk.NextType()
		if next == token.MappingValueType {
			str = Blu(src)
		} else if prev == token.AnchorType || prev == token.AliasType {
			str = Yel(src)
		} else {
			str = Gre(src)
		}
	case token.IntegerType, token.FloatType, token.BoolType:
		str = Mag(src)
	case token.AnchorType, token.AliasType:
		str = Yel(src)
	case token.CommentType:
		str = Whi(src)
	}
	return str
}

// Print YAML object (that don't usually include comments) in color
func PrintYamlColor(yamlObject any) {
	yamlBytes, err := yamlToBytes(yamlObject)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	printYamlBytesColor(yamlBytes)
}

// Print YAML bytes in color, includes comments
// Also prints JSON byte slice in color, but use json.go:printJsonBytesColor() alias instead.
// Caller must ensure yamlBytes is proper YAML/JSON
func printYamlBytesColor(yamlBytes []byte) {
	tokens := lexer.Tokenize(string(yamlBytes))
	if len(tokens) == 0 {
		return
	}
	printOut := []string{}
	//lineNumber := tokens[0].Position.Line
	for _, tk := range tokens {
		lines := strings.Split(tk.Origin, "\n")
		header := ""
		// if allowLineNumber {
		// 	header = fmt.Sprintf("%2d  ", lineNumber)
		// }
		if len(lines) == 1 {
			line := colorizeString(tk, lines[0])
			if len(printOut) == 0 {
				printOut = append(printOut, header+line)
				//lineNumber++
			} else {
				text := printOut[len(printOut)-1]
				printOut[len(printOut)-1] = text + line
			}
		} else {
			header := ""
			for idx, src := range lines {
				// if allowLineNumber {
				// 	header = fmt.Sprintf("%2d  ", lineNumber)
				// }
				line := colorizeString(tk, src)
				if idx == 0 {
					if len(printOut) == 0 {
						printOut = append(printOut, header+line)
						//lineNumber++
					} else {
						text := printOut[len(printOut)-1]
						printOut[len(printOut)-1] = text + line
					}
				} else {
					printOut = append(printOut, fmt.Sprintf("%s%s", header, line))
					//lineNumber++
				}
			}
		}
	}
	fmt.Println(strings.Join(printOut, "\n"))
}
