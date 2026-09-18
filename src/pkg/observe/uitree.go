package observe

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/perf"
)

// Bounds represents the bounding box of a UI element.
type Bounds struct {
	Left   int `json:"left"`
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
}

// Point represents a coordinate.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Element is a filtered, indexed UI element from the hierarchy.
// Bounds and Center are always device pixels (not screenshot preview pixels).
type Element struct {
	Index       int    `json:"index"`
	Handle      string `json:"handle,omitempty"`
	Class       string `json:"class,omitempty"`
	ResourceID  string `json:"resource_id,omitempty"`
	Text        string `json:"text,omitempty"`
	ContentDesc string `json:"content_desc,omitempty"`
	Bounds      Bounds `json:"bounds"`
	Center      Point  `json:"center"`
	Clickable   bool   `json:"clickable,omitempty"`
	Scrollable  bool   `json:"scrollable,omitempty"`
	Focusable   bool   `json:"focusable,omitempty"`
	Enabled     bool   `json:"enabled"`
	Selected    bool   `json:"selected,omitempty"`
	Checked     bool   `json:"checked,omitempty"`
	PackageName string `json:"package,omitempty"`
}

// UITree holds the parsed UI hierarchy.
type UITree struct {
	Package  string         `json:"package,omitempty"`
	StateID  string         `json:"state_id,omitempty"`
	Elements []Element      `json:"elements"`
	Profile  *TimingProfile `json:"profile,omitempty"`
}

// DumpOptions controls how the UI hierarchy is fetched and filtered.
type DumpOptions struct {
	Compressed bool
	Mode       UIMode
	Profile    bool
	Save       bool
}

// xmlNode represents a node in the uiautomator XML dump.
type xmlNode struct {
	XMLName     xml.Name  `xml:"node"`
	Index       string    `xml:"index,attr"`
	Text        string    `xml:"text,attr"`
	ResourceID  string    `xml:"resource-id,attr"`
	Class       string    `xml:"class,attr"`
	Package     string    `xml:"package,attr"`
	ContentDesc string    `xml:"content-desc,attr"`
	Checkable   string    `xml:"checkable,attr"`
	Checked     string    `xml:"checked,attr"`
	Clickable   string    `xml:"clickable,attr"`
	Enabled     string    `xml:"enabled,attr"`
	Focusable   string    `xml:"focusable,attr"`
	Focused     string    `xml:"focused,attr"`
	Scrollable  string    `xml:"scrollable,attr"`
	Selected    string    `xml:"selected,attr"`
	Bounds      string    `xml:"bounds,attr"`
	Children    []xmlNode `xml:"node"`
}

type xmlHierarchy struct {
	XMLName  xml.Name  `xml:"hierarchy"`
	Rotation string    `xml:"rotation,attr"`
	Nodes    []xmlNode `xml:"node"`
}

var boundsRe = regexp.MustCompile(`\[(\d+),(\d+)\]\[(\d+),(\d+)\]`)

func parseBounds(s string) (Bounds, error) {
	m := boundsRe.FindStringSubmatch(s)
	if len(m) != 5 {
		return Bounds{}, fmt.Errorf("invalid bounds format: %s", s)
	}
	l, err := strconv.Atoi(m[1])
	if err != nil {
		return Bounds{}, fmt.Errorf("invalid bounds left value: %w", err)
	}
	t, err := strconv.Atoi(m[2])
	if err != nil {
		return Bounds{}, fmt.Errorf("invalid bounds top value: %w", err)
	}
	r, err := strconv.Atoi(m[3])
	if err != nil {
		return Bounds{}, fmt.Errorf("invalid bounds right value: %w", err)
	}
	b, err := strconv.Atoi(m[4])
	if err != nil {
		return Bounds{}, fmt.Errorf("invalid bounds bottom value: %w", err)
	}
	return Bounds{Left: l, Top: t, Right: r, Bottom: b}, nil
}

// isSignificant returns true if a node is "meaningful" for agent interaction.
// Nodes with only a resource-id that are non-interactive containers (have children)
// are excluded to reduce noise for AI agents.
func isSignificant(n *xmlNode) bool {
	if n.Text != "" || n.ContentDesc != "" {
		return true
	}
	if n.Clickable == "true" || n.Scrollable == "true" {
		return true
	}
	// Keep resource-id nodes only if they are leaf nodes (no children)
	if n.ResourceID != "" && len(n.Children) == 0 {
		return true
	}
	return false
}

func isInteractive(el Element) bool {
	return el.Clickable || el.Scrollable || el.Focusable || el.Text != "" || el.ContentDesc != "" || el.ResourceID != ""
}

// DumpUITree runs a single-shell uiautomator dump and returns the parsed XML.
func DumpUITree(cmd adb.Commander) (*UITree, error) {
	return DumpUITreeOpts(cmd, DumpOptions{Save: true})
}

// DumpUITreeOpts dumps and optionally filters/profiles the UI tree.
func DumpUITreeOpts(cmd adb.Commander, opts DumpOptions) (*UITree, error) {
	clock := perf.Start()
	profile := &TimingProfile{Mode: "combined"}
	if opts.Compressed {
		profile.Mode = "combined-compressed"
	}

	devicePath := fmt.Sprintf("/data/local/tmp/adbclaw_uidump_%d.xml", time.Now().UnixNano())
	dump := "uiautomator dump"
	if opts.Compressed {
		dump += " --compressed"
	}
	// Pass a single adb-shell string. `adb shell sh -c ...` is double-parsed and
	// only runs `uiautomator` on many devices.
	script := fmt.Sprintf("%s %s >/dev/null && cat %s; ec=$?; rm -f %s; exit $ec", dump, devicePath, devicePath, devicePath)

	result, err := cmd.Shell(script)
	profile.ADBCalls = 1
	profile.DumpMs = clock.Lap()
	if err != nil {
		return nil, fmt.Errorf("uiautomator dump failed: %w", err)
	}
	if result.ExitCode != 0 {
		msg := strings.TrimSpace(result.Stderr + result.Stdout)
		if msg == "" {
			msg = fmt.Sprintf("uiautomator dump exit %d", result.ExitCode)
		}
		return nil, fmt.Errorf("uiautomator dump failed: %s", truncate(msg, 200))
	}

	xmlData := result.Stdout
	idx := strings.Index(xmlData, "<?xml")
	if idx < 0 {
		idx = strings.Index(xmlData, "<hierarchy")
	}
	if idx < 0 {
		return nil, fmt.Errorf("uiautomator dump returned no XML data: %s", truncate(xmlData, 200))
	}
	xmlData = xmlData[idx:]

	tree, err := ParseUITree([]byte(xmlData))
	profile.ParseMs = clock.Lap()
	if err != nil {
		return nil, err
	}

	tree = applyUIMode(tree, opts.Mode)
	tree.StateID = NewStateID()
	profile.TotalMs = clock.Total()
	if opts.Profile {
		tree.Profile = profile
	}
	if opts.Save {
		if _, err := SaveSnapshot(commanderSerial(cmd), tree); err != nil {
			// Snapshot cache is an optimization; dump success still stands.
			_ = err
		}
	}
	return tree, nil
}

func applyUIMode(tree *UITree, mode UIMode) *UITree {
	mode = normalizeUIMode(mode)
	if tree == nil || mode == UIModeFull {
		assignHandles(tree)
		return tree
	}

	out := make([]Element, 0, len(tree.Elements))
	for _, el := range tree.Elements {
		if mode == UIModeInteractive && !isInteractive(el) {
			continue
		}
		if mode == UIModeCompact || mode == UIModeRealtime {
			el.Class = ""
			el.PackageName = ""
			if mode == UIModeRealtime && !el.Clickable && !el.Scrollable && el.Text == "" && el.ContentDesc == "" && el.ResourceID == "" {
				continue
			}
		}
		out = append(out, el)
	}
	for i := range out {
		out[i].Index = i
	}
	tree.Elements = out
	assignHandles(tree)
	return tree
}

func assignHandles(tree *UITree) {
	if tree == nil {
		return
	}
	for i := range tree.Elements {
		tree.Elements[i].Handle = fmt.Sprintf("e%d", tree.Elements[i].Index)
	}
}

// ParseUITree parses uiautomator XML into a UITree with indexed elements.
func ParseUITree(data []byte) (*UITree, error) {
	var h xmlHierarchy
	if err := xml.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("xml parse error: %w", err)
	}

	var elements []Element
	var walk func(nodes []xmlNode)
	walk = func(nodes []xmlNode) {
		for i := range nodes {
			n := &nodes[i]
			if isSignificant(n) {
				bounds, err := parseBounds(n.Bounds)
				if err == nil {
					el := Element{
						Index:       len(elements),
						Class:       n.Class,
						ResourceID:  n.ResourceID,
						Text:        n.Text,
						ContentDesc: n.ContentDesc,
						Bounds:      bounds,
						Center: Point{
							X: (bounds.Left + bounds.Right) / 2,
							Y: (bounds.Top + bounds.Bottom) / 2,
						},
						Clickable:   n.Clickable == "true",
						Scrollable:  n.Scrollable == "true",
						Focusable:   n.Focusable == "true",
						Enabled:     n.Enabled == "true",
						Selected:    n.Selected == "true",
						Checked:     n.Checked == "true",
						PackageName: n.Package,
					}
					elements = append(elements, el)
				}
			}
			walk(n.Children)
		}
	}
	walk(h.Nodes)

	pkg, elements := hoistPackage(elements)
	tree := &UITree{Package: pkg, Elements: elements}
	assignHandles(tree)
	return tree, nil
}

// hoistPackage lifts the most common package name to the tree level and
// clears it on matching elements so JSON is not repeated per node.
func hoistPackage(elements []Element) (string, []Element) {
	counts := map[string]int{}
	for _, el := range elements {
		if el.PackageName != "" {
			counts[el.PackageName]++
		}
	}
	best, bestN := "", 0
	for p, n := range counts {
		if n > bestN {
			best, bestN = p, n
		}
	}
	if best == "" {
		return "", elements
	}
	out := make([]Element, len(elements))
	for i, el := range elements {
		out[i] = el
		if el.PackageName == best {
			out[i].PackageName = ""
		}
	}
	return best, out
}

// FindByIndex returns the element at the given index.
func (t *UITree) FindByIndex(index int) (*Element, error) {
	if index < 0 || index >= len(t.Elements) {
		return nil, fmt.Errorf("index %d out of range (0-%d)", index, len(t.Elements)-1)
	}
	return &t.Elements[index], nil
}

// FindByHandle returns the element with the given handle or index string.
func (t *UITree) FindByHandle(handle string) (*Element, error) {
	handle = strings.TrimSpace(handle)
	if handle == "" {
		return nil, fmt.Errorf("empty element handle")
	}
	for i := range t.Elements {
		if t.Elements[i].Handle == handle {
			return &t.Elements[i], nil
		}
	}
	if strings.HasPrefix(handle, "e") {
		if n, err := strconv.Atoi(handle[1:]); err == nil {
			return t.FindByIndex(n)
		}
	}
	if n, err := strconv.Atoi(handle); err == nil {
		return t.FindByIndex(n)
	}
	return nil, fmt.Errorf("no element with handle %q", handle)
}

// FindByText returns elements whose text contains the query (case-insensitive).
func (t *UITree) FindByText(query string) []Element {
	query = strings.ToLower(query)
	var results []Element
	for _, el := range t.Elements {
		if strings.Contains(strings.ToLower(el.Text), query) ||
			strings.Contains(strings.ToLower(el.ContentDesc), query) {
			results = append(results, el)
		}
	}
	return results
}

// FindByID returns elements whose resource-id contains the query.
func (t *UITree) FindByID(query string) []Element {
	query = strings.ToLower(query)
	var results []Element
	for _, el := range t.Elements {
		if strings.Contains(strings.ToLower(el.ResourceID), query) {
			results = append(results, el)
		}
	}
	return results
}

// Hash returns a stable fingerprint of interactive nodes for delta detection.
func (t *UITree) Hash() string {
	if t == nil {
		return ""
	}
	var b strings.Builder
	for _, el := range t.Elements {
		fmt.Fprintf(&b, "%s|%s|%s|%d,%d|%t|%t;", el.Text, el.ContentDesc, el.ResourceID, el.Center.X, el.Center.Y, el.Clickable, el.Scrollable)
	}
	sum := sha1Short(b.String())
	return sum
}

func sha1Short(s string) string {
	// local import avoided circular issues — keep tiny helper here via fmt only
	h := 2166136261
	for i := 0; i < len(s); i++ {
		h ^= int(s[i])
		h *= 16777619
	}
	if h < 0 {
		h = -h
	}
	return fmt.Sprintf("%08x", h)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
