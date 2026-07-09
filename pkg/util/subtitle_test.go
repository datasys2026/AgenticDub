package util

import (
	"fmt"
	"reflect"
	"testing"
)

func TestSplitTextSentences(t *testing.T) {
	origin_sentence := "Now, I'm Ryan D'Aris, founder and CEO of flowstate.com"
	sentences := SplitTextSentences(origin_sentence, 55)
	fmt.Println("origin sentence:", origin_sentence)
	for i, sentence := range sentences {
		fmt.Printf("sentence %d, got '%s'\n", i, sentence)
	}

	// 期望的结果：应该保持为一个完整句子，因为有效字符数小于70
	if len(sentences) != 1 {
		t.Errorf("Expected 1 sentence, got %d sentences", len(sentences))
	}
}

func TestSplitTextSentencesMergesStandaloneQuotes(t *testing.T) {
	got := SplitTextSentences("He said hello. \" Next sentence.", 10)
	want := []string{"He said hello.\"", "Next sentence."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}
