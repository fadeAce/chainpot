package chainpot

type trieNode struct {
	children map[byte]*trieNode
	isEnd    bool
}

func newTrieNode() *trieNode {
	return &trieNode{children: make(map[byte]*trieNode), isEnd: false}
}

type trie struct {
	root *trieNode
}

type Pot struct {
	trie *trie
}

func newtrie() *trie {
	return &trie{root: newTrieNode()}
}
func NewPot() *Pot {
	return &Pot{trie: newtrie()}
}

func (trie *trie) insert(word []byte) {
	node := trie.root
	for i := 0; i < len(word); i++ {
		_, ok := node.children[word[i]]
		if !ok {
			node.children[word[i]] = newTrieNode()
		}
		node = node.children[word[i]]
	}
	node.isEnd = true
}

func (trie *trie) search(word []byte) bool {
	node := trie.root
	for i := 0; i < len(word); i++ {
		_, ok := node.children[word[i]]
		if !ok {
			return false
		}
		node = node.children[word[i]]
	}
	return node.isEnd
}

func (trie *trie) startsWith(prefix []byte) bool {
	node := trie.root
	for i := 0; i < len(prefix); i++ {
		_, ok := node.children[prefix[i]]
		if !ok {
			return false
		}
		node = node.children[prefix[i]]
	}
	return true
}

func (p *Pot) Search(str string) bool {
	return p.trie.search([]byte(str))
}

func (p *Pot) Insert(str string) {
	p.trie.insert([]byte(str))
}
