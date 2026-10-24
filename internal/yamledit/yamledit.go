// Package yamledit applies an edited config, as plain values, onto the YAML
// file it came from: the keys kept keep their place and their comments,
// only the values that changed are written, the keys removed go away.
package yamledit

import (
	"bytes"

	"go.yaml.in/yaml/v3"

	"github.com/davidp57/process-monitor/internal/i18n"
)

// Read gives the values of a config file, as the interface edits them.
func Read(data []byte) (map[string]any, error) {
	doc := map[string]any{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// Apply returns the file data with its values replaced by doc.
func Apply(data []byte, doc map[string]any) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	var want yaml.Node
	if err := want.Encode(doc); err != nil {
		return nil, err
	}
	if root.Kind == 0 { // an empty file
		root = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{&want}}
	} else if len(root.Content) != 1 || root.Content[0].Kind != yaml.MappingNode {
		return nil, i18n.Errorf("gui.not_a_mapping")
	} else {
		merge(root.Content[0], &want)
	}
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		return nil, err
	}
	return out.Bytes(), enc.Close()
}

// merge makes dst say what src says, keeping what dst already had right:
// a mapping keeps its keys in their order with their comments, a value that
// did not change keeps its node (its quotes, its style, its comments).
func merge(dst, src *yaml.Node) {
	if dst.Kind == yaml.MappingNode && src.Kind == yaml.MappingNode {
		wanted := map[string]*yaml.Node{}
		var order []string
		for i := 0; i+1 < len(src.Content); i += 2 {
			wanted[src.Content[i].Value] = src.Content[i+1]
			order = append(order, src.Content[i].Value)
		}
		var kept []*yaml.Node
		have := map[string]bool{}
		for i := 0; i+1 < len(dst.Content); i += 2 {
			key, val := dst.Content[i], dst.Content[i+1]
			w, ok := wanted[key.Value]
			if !ok {
				continue // removed
			}
			merge(val, w)
			kept = append(kept, key, val)
			have[key.Value] = true
		}
		for _, k := range order {
			if !have[k] {
				kept = append(kept, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}, wanted[k])
			}
		}
		// "DEV: {}" that gets keys is written as a block, like the rest.
		if len(dst.Content) == 0 && len(kept) > 0 {
			dst.Style = src.Style
		}
		dst.Content = kept
		return
	}
	if same(dst, src) {
		return
	}
	head, line, foot := dst.HeadComment, dst.LineComment, dst.FootComment
	style := dst.Style
	*dst = *src
	dst.HeadComment, dst.LineComment, dst.FootComment = head, line, foot
	// A quoted path stays quoted: backslashes read the same either way.
	if dst.Kind == yaml.ScalarNode && style&(yaml.SingleQuotedStyle|yaml.DoubleQuotedStyle) != 0 && src.Tag == "!!str" {
		dst.Style = style
	}
}

// same reports whether two nodes say the same thing.
func same(a, b *yaml.Node) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case yaml.ScalarNode:
		return a.Value == b.Value && a.ShortTag() == b.ShortTag()
	case yaml.SequenceNode, yaml.MappingNode:
		if len(a.Content) != len(b.Content) {
			return false
		}
		for i := range a.Content {
			if !same(a.Content[i], b.Content[i]) {
				return false
			}
		}
		return true
	}
	return false
}
