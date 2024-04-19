package jieba

import (
	"github.com/jjonline/share-mod-lib/jieba/deps/cppjieba"
	"github.com/jjonline/share-mod-lib/jieba/deps/limonp"
	"github.com/jjonline/share-mod-lib/jieba/dict"
)

func init() {
	dict.Init()
	limonp.Init()
	cppjieba.Init()
}
