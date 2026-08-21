package detector

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// VDFNode represents a single key-value or parent section node in a Valve KeyValues/VDF tree.
type VDFNode struct {
	Key      string
	Value    string
	Children []*VDFNode
}

// Get finds the direct child with matching key (case-insensitive).
func (n *VDFNode) Get(key string) *VDFNode {
	if n == nil {
		return nil
	}
	for _, child := range n.Children {
		if strings.EqualFold(child.Key, key) {
			return child
		}
	}
	return nil
}

// GetString retrieves the string value of a child or path (slash-separated e.g. "AppState/name" or "name").
func (n *VDFNode) GetString(path string) string {
	if n == nil {
		return ""
	}
	parts := strings.Split(path, "/")
	curr := n
	for _, p := range parts {
		if curr == nil {
			return ""
		}
		if strings.EqualFold(curr.Key, p) && len(parts) > 1 && curr == n {
			continue
		}
		curr = curr.Get(p)
	}
	if curr != nil {
		return curr.Value
	}
	return ""
}

// GetInt64 retrieves an integer value at the specified path.
func (n *VDFNode) GetInt64(path string) int64 {
	valStr := n.GetString(path)
	if valStr == "" {
		return 0
	}
	v, _ := strconv.ParseInt(valStr, 10, 64)
	return v
}

// GetSubKeys returns all direct children as a map of key -> value.
func (n *VDFNode) GetSubKeys() map[string]string {
	res := make(map[string]string)
	if n == nil {
		return res
	}
	for _, child := range n.Children {
		if len(child.Children) == 0 {
			res[child.Key] = child.Value
		}
	}
	return res
}

// ParseVDFFile parses a VDF/ACF file from disk.
func ParseVDFFile(path string) (*VDFNode, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read vdf file: %w", err)
	}
	return ParseVDF(bytes.NewReader(data))
}

// ParseVDF parses a VDF/ACF formatted text stream into a root VDFNode.
func ParseVDF(r io.Reader) (*VDFNode, error) {
	tokens, err := tokenizeVDF(r)
	if err != nil {
		return nil, err
	}

	root := &VDFNode{Key: "root"}
	idx := 0
	for idx < len(tokens) {
		child, nextIdx, err := parseVDFNode(tokens, idx)
		if err != nil {
			return nil, err
		}
		if child != nil {
			root.Children = append(root.Children, child)
		}
		idx = nextIdx
	}

	// If there is only one top-level section (like "AppState"), return it as primary node
	if len(root.Children) == 1 {
		return root.Children[0], nil
	}

	return root, nil
}

func parseVDFNode(tokens []string, start int) (*VDFNode, int, error) {
	if start >= len(tokens) {
		return nil, start, nil
	}

	key := tokens[start]
	if key == "}" {
		return nil, start + 1, nil
	}

	if start+1 >= len(tokens) {
		return &VDFNode{Key: key}, start + 1, nil
	}

	next := tokens[start+1]
	if next == "{" {
		// Parent section
		node := &VDFNode{Key: key}
		idx := start + 2
		for idx < len(tokens) {
			if tokens[idx] == "}" {
				return node, idx + 1, nil
			}
			child, nextIdx, err := parseVDFNode(tokens, idx)
			if err != nil {
				return nil, nextIdx, err
			}
			if child != nil {
				node.Children = append(node.Children, child)
			}
			idx = nextIdx
		}
		return node, idx, nil
	}

	// Key-value pair
	return &VDFNode{Key: key, Value: next}, start + 2, nil
}

func tokenizeVDF(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	var tokens []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Strip inline comments
		if idx := strings.Index(line, "//"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		inQuote := false
		var curToken strings.Builder
		runes := []rune(line)

		for i := 0; i < len(runes); i++ {
			ch := runes[i]

			if ch == '"' {
				if inQuote {
					tokens = append(tokens, curToken.String())
					curToken.Reset()
					inQuote = false
				} else {
					inQuote = true
				}
				continue
			}

			if inQuote {
				curToken.WriteRune(ch)
				continue
			}

			if ch == '{' || ch == '}' {
				if curToken.Len() > 0 {
					tokens = append(tokens, curToken.String())
					curToken.Reset()
				}
				tokens = append(tokens, string(ch))
				continue
			}

			if unicode.IsSpace(ch) {
				if curToken.Len() > 0 {
					tokens = append(tokens, curToken.String())
					curToken.Reset()
				}
				continue
			}

			curToken.WriteRune(ch)
		}

		if inQuote {
			tokens = append(tokens, curToken.String())
		} else if curToken.Len() > 0 {
			tokens = append(tokens, curToken.String())
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan vdf: %w", err)
	}

	return tokens, nil
}
