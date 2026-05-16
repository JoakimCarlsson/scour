package engines

import "testing"

func TestUnwrapBingRedirect(t *testing.T) {
	cases := []struct{ in, want string }{
		// a1aHR0cHM6Ly93d3cucG9ybmh1Yi5jb20v decodes to https://www.pornhub.com/
		{
			"https://www.bing.com/ck/a?%21=&fclid=abc&u=a1aHR0cHM6Ly93d3cucG9ybmh1Yi5jb20v&ver=2",
			"https://www.pornhub.com/",
		},
		// a1aHR0cHM6Ly9nby5kZXYv decodes to https://go.dev/
		{
			"https://www.bing.com/ck/a?p=x&u=a1aHR0cHM6Ly9nby5kZXYv&ver=2",
			"https://go.dev/",
		},
		// Non-bing URL passes through.
		{"https://example.com/foo", "https://example.com/foo"},
		// bing.com but not ck/a passes through.
		{"https://www.bing.com/search?q=x", "https://www.bing.com/search?q=x"},
		// u param without a1 prefix passes through.
		{"https://www.bing.com/ck/a?u=foo&ver=2", "https://www.bing.com/ck/a?u=foo&ver=2"},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			got := unwrapBingRedirect(c.in)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
