//关键词检索trie树，目前是用于敏感词过滤
package trie

import "fmt"

//trie树节点
type TrieElement struct {
	isLeaf   bool
	children map[rune]*TrieElement
}

//trie树
type Trie struct {
	tree TrieElement
}

func New() Trie {
	return Trie{TrieElement{false, make(map[rune]*TrieElement)}}
}

//添加关键词
func (t *Trie) AddElement(key string) {
	curr := &t.tree
	for _, char := range key {
		elem, found := curr.children[char]
		if !found {
			elem = &TrieElement{false, make(map[rune]*TrieElement)}
			curr.children[char] = elem
		}
		curr = elem
	}
	curr.isLeaf = true
}

func printSubTree(level uint, t *TrieElement) {
	for key, child := range t.children {
		tabs := ""
		for i := uint(0); i < level; i++ {
			tabs += " "
		}
		leaf := ""
		if child.isLeaf {
			leaf = "*"
		}
		fmt.Printf("%s%s%s\n", tabs, leaf, string(key))
		printSubTree(level+1, child)

	}
}

//打印trie树结构
func (t *Trie) PrintKeys() {
	printSubTree(0, &t.tree)
}

func (t *Trie) searchHelper(text string) (found bool, key string) {
	curr := &t.tree
	i := 0
	maxLen := 0
	var char rune
	for i, char = range text {
		elem, found := curr.children[char]
		if !found {
			break
		} else {
			if elem.isLeaf {
				maxLen = i + len(string(char))
			}
			curr = elem
		}
	}
	if maxLen == 0 {
		return false, ""
	} else {
		return true, text[0:maxLen]
	}
}

//搜索文本中能够匹配上的第一个关键词,返回其位置和关键词
func (t *Trie) Search(text string) (pos int, key string) {
	for i, _ := range text {
		found, key := t.searchHelper(text[i:])
		if found {
			return i, key
		}
	}
	return -1, ""
}

//替换文本中所有匹配上的敏感词
//参数：
//	num: 敏感词数量
//	replaced: 替换后的文本
func (t *Trie) Replace(text string) (num int, replaced string) {
	b := make([]byte, 0, len(text))
	num = 0
	for i := 0; i < len(text); {
		found, key := t.searchHelper(text[i:])
		if found {
			num++
			for _, _ = range key {
				b = append(b, '*')
			}
			i += len(key)
		} else {
			b = append(b, text[i])
			i++
		}
	}
	return num, string(b)
}
