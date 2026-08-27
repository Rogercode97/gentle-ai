package main

import (
	"io/ioutil"
	"strings"
)

func main() {
	content, _ := ioutil.ReadFile("internal/components/sdd/antigravity_sdd_agents.go")
	s := string(content)

	oldMerge := `hooksWrite, err := mergeJSONFile(hooksPath, antigravitySddAgentsHooksJSON())`
	
	newMerge := `hooksBase, err := readFileOrEmpty(hooksPath)
		if err != nil {
			return false, nil, fmt.Errorf("read hooks: %w", err)
		}
		var baseBytes []byte
		if hooksBase != "" {
			baseBytes = []byte(hooksBase)
		}
		hooksWrite, err := mergeJSONFileContents(hooksPath, baseBytes, antigravitySddAgentsHooksJSON())`

	s = strings.Replace(s, oldMerge, newMerge, 1)

	ioutil.WriteFile("internal/components/sdd/antigravity_sdd_agents.go", []byte(s), 0644)
}
