package ini

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// ElementType indicates the type of item within an INI section.
type ElementType int

const (
	// ElemBlank represents an empty line.
	ElemBlank ElementType = iota
	// ElemComment represents a full-line comment starting with '#' or ';'.
	ElemComment
	// ElemKeyValue represents a key-value pair.
	ElemKeyValue
)

// Element represents a single line / item within a section, preserving order and comments.
type Element struct {
	Type      ElementType
	Raw       string // Raw line for blank lines and comments
	Key       string // Original key casing
	Value     string // Stored raw value
	Separator string // e.g. "=" or " = "
}

// KeyValue represents an extracted key and value pair.
type KeyValue struct {
	Key   string
	Value string
}

// Section represents a section in an INI file.
type Section struct {
	mu       sync.RWMutex
	name     string
	elements []Element
	keyIndex map[string]int // lowercase key -> index in elements
}

func newSection(name string) *Section {
	return &Section{
		name:     name,
		elements: make([]Element, 0),
		keyIndex: make(map[string]int),
	}
}

// Name returns the section name with its original casing.
func (s *Section) Name() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.name
}

// SetName updates the section name.
func (s *Section) SetName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.name = strings.TrimSpace(name)
}

// Get returns the value for key with case-insensitive matching.
// Surrounding matching single or double quotes are automatically stripped.
func (s *Section) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx, ok := s.keyIndex[strings.ToLower(strings.TrimSpace(key))]
	if !ok {
		return "", false
	}
	return unquote(s.elements[idx].Value), true
}

// GetRaw returns the verbatim stored value for key without stripping quotes.
func (s *Section) GetRaw(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx, ok := s.keyIndex[strings.ToLower(strings.TrimSpace(key))]
	if !ok {
		return "", false
	}
	return s.elements[idx].Value, true
}

// GetString returns the value for key or defaultVal if the key is missing or empty.
func (s *Section) GetString(key string, defaultVal string) string {
	if val, ok := s.Get(key); ok && val != "" {
		return val
	}
	return defaultVal
}

// GetBool returns the boolean interpretation of key's value.
// Truthy values: "1", "true", "t", "yes", "y", "on" (case-insensitive).
// Falsy values: "0", "false", "f", "no", "n", "off" (case-insensitive).
func (s *Section) GetBool(key string) (bool, bool) {
	val, ok := s.Get(key)
	if !ok {
		return false, false
	}
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "1", "true", "t", "yes", "y", "on":
		return true, true
	case "0", "false", "f", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}

// GetInt returns the integer representation of key's value.
func (s *Section) GetInt(key string) (int, bool) {
	val, ok := s.Get(key)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		return 0, false
	}
	return n, true
}

// HasKey reports whether the section contains key (case-insensitive).
func (s *Section) HasKey(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.keyIndex[strings.ToLower(strings.TrimSpace(key))]
	return ok
}

// Set sets a key-value pair. If the key exists, its value is updated in place,
// preserving the original separator formatting and surrounding comments.
// If it does not exist, it is appended with a standard "=" separator.
func (s *Section) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	trimmedKey := strings.TrimSpace(key)
	lower := strings.ToLower(trimmedKey)

	if idx, ok := s.keyIndex[lower]; ok {
		s.elements[idx].Value = value
		s.elements[idx].Key = trimmedKey
		return
	}

	elem := Element{
		Type:      ElemKeyValue,
		Key:       trimmedKey,
		Value:     value,
		Separator: "=",
	}
	s.elements = append(s.elements, elem)
	s.keyIndex[lower] = len(s.elements) - 1
}

// SetBool sets a boolean value. If the key already has a "true"/"false" value,
// it preserves that word style; otherwise it defaults to "1" or "0".
func (s *Section) SetBool(key string, val bool) {
	s.mu.RLock()
	existing, ok := s.keyIndex[strings.ToLower(strings.TrimSpace(key))]
	var wordStyle bool
	if ok {
		cur := strings.ToLower(strings.TrimSpace(s.elements[existing].Value))
		if cur == "true" || cur == "false" {
			wordStyle = true
		}
	}
	s.mu.RUnlock()

	if wordStyle {
		if val {
			s.Set(key, "true")
		} else {
			s.Set(key, "false")
		}
	} else {
		if val {
			s.Set(key, "1")
		} else {
			s.Set(key, "0")
		}
	}
}

// SetInt sets an integer value.
func (s *Section) SetInt(key string, val int) {
	s.Set(key, strconv.Itoa(val))
}

// Delete removes a key and its value from the section.
func (s *Section) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	lower := strings.ToLower(strings.TrimSpace(key))
	idx, ok := s.keyIndex[lower]
	if !ok {
		return false
	}

	s.elements = append(s.elements[:idx], s.elements[idx+1:]...)
	s.rebuildKeyIndex()
	return true
}

// Keys returns all keys present in the section in file order.
func (s *Section) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.keyIndex))
	for _, elem := range s.elements {
		if elem.Type == ElemKeyValue {
			keys = append(keys, elem.Key)
		}
	}
	return keys
}

// KeyValues returns all key-value pairs in the section in file order.
func (s *Section) KeyValues() []KeyValue {
	s.mu.RLock()
	defer s.mu.RUnlock()

	kvs := make([]KeyValue, 0, len(s.keyIndex))
	for _, elem := range s.elements {
		if elem.Type == ElemKeyValue {
			kvs = append(kvs, KeyValue{
				Key:   elem.Key,
				Value: unquote(elem.Value),
			})
		}
	}
	return kvs
}

// Items returns a map of all key-value pairs in the section (keys lowercased).
func (s *Section) Items() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make(map[string]string, len(s.keyIndex))
	for _, elem := range s.elements {
		if elem.Type == ElemKeyValue {
			items[elem.Key] = unquote(elem.Value)
		}
	}
	return items
}

func (s *Section) rebuildKeyIndex() {
	s.keyIndex = make(map[string]int)
	for i, elem := range s.elements {
		if elem.Type == ElemKeyValue {
			s.keyIndex[strings.ToLower(elem.Key)] = i
		}
	}
}

// File represents a parsed INI document.
type File struct {
	mu           sync.RWMutex
	sections     []*Section
	sectionIndex map[string]int // lowercase section name -> index in sections
}

// NewFile creates a new empty INI document with an initialized default section.
func NewFile() *File {
	f := &File{
		sections:     make([]*Section, 0),
		sectionIndex: make(map[string]int),
	}
	defaultSec := newSection("")
	f.sections = append(f.sections, defaultSec)
	f.sectionIndex[""] = 0
	return f
}

// Section returns the section with the given name (case-insensitive).
// If the section does not exist, it is created and appended to the file.
func (f *File) Section(name string) *Section {
	f.mu.Lock()
	defer f.mu.Unlock()

	trimmed := strings.TrimSpace(name)
	lower := strings.ToLower(trimmed)

	if idx, ok := f.sectionIndex[lower]; ok {
		return f.sections[idx]
	}

	sec := newSection(trimmed)
	f.sections = append(f.sections, sec)
	f.sectionIndex[lower] = len(f.sections) - 1
	return sec
}

// GetSection returns the section with the given name (case-insensitive), or (nil, false).
func (f *File) GetSection(name string) (*Section, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	lower := strings.ToLower(strings.TrimSpace(name))
	idx, ok := f.sectionIndex[lower]
	if !ok {
		return nil, false
	}
	return f.sections[idx], true
}

// HasSection reports whether the file contains the named section (case-insensitive).
func (f *File) HasSection(name string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, ok := f.sectionIndex[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

// DeleteSection removes a section by name.
func (f *File) DeleteSection(name string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	lower := strings.ToLower(strings.TrimSpace(name))
	idx, ok := f.sectionIndex[lower]
	if !ok {
		return false
	}

	f.sections = append(f.sections[:idx], f.sections[idx+1:]...)
	f.rebuildSectionIndex()
	return true
}

// Sections returns all sections in file order.
func (f *File) Sections() []*Section {
	f.mu.RLock()
	defer f.mu.RUnlock()

	res := make([]*Section, len(f.sections))
	copy(res, f.sections)
	return res
}

// SectionNames returns the names of all non-empty sections in file order.
func (f *File) SectionNames() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var names []string
	for _, sec := range f.sections {
		if sec.name == "" && len(sec.elements) == 0 {
			continue
		}
		names = append(names, sec.name)
	}
	return names
}

// Get retrieves a key's value from the specified section.
func (f *File) Get(section, key string) (string, bool) {
	sec, ok := f.GetSection(section)
	if !ok {
		return "", false
	}
	return sec.Get(key)
}

// GetString retrieves a key's value from the specified section or returns defaultVal.
func (f *File) GetString(section, key, defaultVal string) string {
	sec, ok := f.GetSection(section)
	if !ok {
		return defaultVal
	}
	return sec.GetString(key, defaultVal)
}

// GetBool retrieves a boolean value from the specified section.
func (f *File) GetBool(section, key string) (bool, bool) {
	sec, ok := f.GetSection(section)
	if !ok {
		return false, false
	}
	return sec.GetBool(key)
}

// GetInt retrieves an integer value from the specified section.
func (f *File) GetInt(section, key string) (int, bool) {
	sec, ok := f.GetSection(section)
	if !ok {
		return 0, false
	}
	return sec.GetInt(key)
}

// Set sets a key-value pair in the specified section, creating the section if necessary.
func (f *File) Set(section, key, value string) {
	f.Section(section).Set(key, value)
}

// SetBool sets a boolean value in the specified section.
func (f *File) SetBool(section, key string, val bool) {
	f.Section(section).SetBool(key, val)
}

// SetInt sets an integer value in the specified section.
func (f *File) SetInt(section, key string, val int) {
	f.Section(section).SetInt(key, val)
}

// Delete removes a key from the specified section.
func (f *File) Delete(section, key string) bool {
	sec, ok := f.GetSection(section)
	if !ok {
		return false
	}
	return sec.Delete(key)
}

func (f *File) rebuildSectionIndex() {
	f.sectionIndex = make(map[string]int)
	for i, sec := range f.sections {
		f.sectionIndex[strings.ToLower(sec.name)] = i
	}
}

// Load parses INI data from an io.Reader.
func Load(r io.Reader) (*File, error) {
	f := NewFile()

	scanner := bufio.NewScanner(r)
	// Support long lines up to 10MB
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	currSec := f.Section("")

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// 1. Blank line
		if trimmed == "" {
			currSec.elements = append(currSec.elements, Element{
				Type: ElemBlank,
				Raw:  line,
			})
			continue
		}

		// 2. Full-line comment
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			currSec.elements = append(currSec.elements, Element{
				Type: ElemComment,
				Raw:  line,
			})
			continue
		}

		// 3. Section header: [section_name]
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			secName := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			currSec = f.Section(secName)
			continue
		}

		// 4. Key-value line: key = value
		if eqIdx := strings.Index(line, "="); eqIdx != -1 {
			rawKeyPart := line[:eqIdx]
			rawValPart := line[eqIdx+1:]

			key := strings.TrimSpace(rawKeyPart)
			val := strings.TrimSpace(rawValPart)

			sep := "="
			hasLeftSpace := strings.HasSuffix(rawKeyPart, " ") || strings.HasSuffix(rawKeyPart, "\t")
			hasRightSpace := strings.HasPrefix(rawValPart, " ") || strings.HasPrefix(rawValPart, "\t")
			if hasLeftSpace && hasRightSpace {
				sep = " = "
			} else if hasLeftSpace {
				sep = " ="
			} else if hasRightSpace {
				sep = "= "
			}

			lowerKey := strings.ToLower(key)
			if existingIdx, ok := currSec.keyIndex[lowerKey]; ok {
				// Last definition wins, but update in-place
				currSec.elements[existingIdx].Key = key
				currSec.elements[existingIdx].Value = val
				currSec.elements[existingIdx].Separator = sep
			} else {
				currSec.elements = append(currSec.elements, Element{
					Type:      ElemKeyValue,
					Key:       key,
					Value:     val,
					Separator: sep,
				})
				currSec.keyIndex[lowerKey] = len(currSec.elements) - 1
			}
			continue
		}

		// 5. Fallback for unparseable line: treat as comment to preserve it
		currSec.elements = append(currSec.elements, Element{
			Type: ElemComment,
			Raw:  line,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read ini: %w", err)
	}

	return f, nil
}

// LoadBytes parses INI data from a byte slice.
func LoadBytes(data []byte) (*File, error) {
	return Load(bytes.NewReader(data))
}

// LoadString parses INI data from a string.
func LoadString(s string) (*File, error) {
	return Load(strings.NewReader(s))
}

// LoadFile parses an INI file from disk.
func LoadFile(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Load(f)
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.n += int64(n)
	return n, err
}

// WriteTo writes the INI document to an io.Writer, implementing io.WriterTo.
func (f *File) WriteTo(w io.Writer) (int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	cw := &countingWriter{w: w}
	bw := bufio.NewWriter(cw)
	writtenAnything := false
	lastWasBlank := false

	for _, sec := range f.sections {
		sec.mu.RLock()
		isDefault := sec.name == ""
		hasElements := len(sec.elements) > 0

		// Skip empty default section
		if isDefault && !hasElements {
			sec.mu.RUnlock()
			continue
		}

		// Separate sections with a newline if preceding output didn't end in blank
		if writtenAnything && !isDefault && !lastWasBlank {
			if err := bw.WriteByte('\n'); err != nil {
				sec.mu.RUnlock()
				_ = bw.Flush()
				return cw.n, err
			}
		}

		// Write section header if not default section
		if !isDefault {
			if _, err := bw.WriteString(fmt.Sprintf("[%s]\n", sec.name)); err != nil {
				sec.mu.RUnlock()
				_ = bw.Flush()
				return cw.n, err
			}
			writtenAnything = true
			lastWasBlank = false
		}

		// Write section elements
		for _, elem := range sec.elements {
			switch elem.Type {
			case ElemBlank:
				if err := bw.WriteByte('\n'); err != nil {
					sec.mu.RUnlock()
					_ = bw.Flush()
					return cw.n, err
				}
				lastWasBlank = true
			case ElemComment:
				if _, err := bw.WriteString(elem.Raw + "\n"); err != nil {
					sec.mu.RUnlock()
					_ = bw.Flush()
					return cw.n, err
				}
				lastWasBlank = false
			case ElemKeyValue:
				sep := elem.Separator
				if sep == "" {
					sep = "="
				}
				line := fmt.Sprintf("%s%s%s\n", elem.Key, sep, elem.Value)
				if _, err := bw.WriteString(line); err != nil {
					sec.mu.RUnlock()
					_ = bw.Flush()
					return cw.n, err
				}
				lastWasBlank = false
			}
			writtenAnything = true
		}
		sec.mu.RUnlock()
	}

	err := bw.Flush()
	return cw.n, err
}

// Bytes returns the INI document serialized as bytes.
func (f *File) Bytes() []byte {
	var buf bytes.Buffer
	_, _ = f.WriteTo(&buf)
	return buf.Bytes()
}

// String returns the INI document serialized as a string.
func (f *File) String() string {
	return string(f.Bytes())
}

// SaveFile writes the INI document atomically to the target file path.
func (f *File) SaveFile(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	data := f.Bytes()

	// Atomic write using a temporary file in the same directory
	tmpFile, err := os.CreateTemp(dir, ".gamux-ini-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath) // Clean up if rename didn't take place
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Preserve permissions if target file already exists
	perm := os.FileMode(0644)
	if fi, err := os.Stat(path); err == nil {
		perm = fi.Mode().Perm()
	}
	_ = os.Chmod(tmpPath, perm)

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file to %s: %w", path, err)
	}

	return nil
}

func unquote(s string) string {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) >= 2 {
		first := trimmed[0]
		last := trimmed[len(trimmed)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return trimmed[1 : len(trimmed)-1]
		}
	}
	return trimmed
}
