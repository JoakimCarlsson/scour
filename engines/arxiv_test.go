package engines

import "testing"

const arxivFixture = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <title>Attention Is All You Need</title>
    <summary>The dominant sequence transduction models...</summary>
    <published>2017-06-12T17:57:34Z</published>
    <link href="https://arxiv.org/abs/1706.03762v7" rel="alternate" type="text/html"/>
    <link href="https://arxiv.org/pdf/1706.03762v7" rel="related" type="application/pdf"/>
    <author><name>Ashish Vaswani</name></author>
    <author><name>Noam Shazeer</name></author>
  </entry>
  <entry>
    <title>BERT: Pre-training of Deep Bidirectional Transformers</title>
    <summary>We introduce a new language representation model called BERT</summary>
    <published>2018-10-11T00:50:01Z</published>
    <link href="https://arxiv.org/abs/1810.04805v2" rel="alternate" type="text/html"/>
    <author><name>Jacob Devlin</name></author>
  </entry>
</feed>`

func TestParseArxiv(t *testing.T) {
	resp, err := parseArxiv([]byte(arxivFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	if resp.Results[0].URL != "https://arxiv.org/abs/1706.03762v7" {
		t.Errorf("url=%q", resp.Results[0].URL)
	}
	if resp.Results[0].Extras[ExtraAuthor] != "Ashish Vaswani, Noam Shazeer" {
		t.Errorf("author=%q", resp.Results[0].Extras[ExtraAuthor])
	}
	if resp.Results[0].Extras[ExtraPublishedAt] != "2017-06-12T17:57:34Z" {
		t.Errorf("published=%q", resp.Results[0].Extras[ExtraPublishedAt])
	}
}

func TestParseArxivEmpty(t *testing.T) {
	if _, err := parseArxiv([]byte(`<feed xmlns="http://www.w3.org/2005/Atom"></feed>`)); err == nil {
		t.Fatal("expected error")
	}
}
