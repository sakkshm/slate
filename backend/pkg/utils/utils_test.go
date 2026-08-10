package utils

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Weather.io", "weather-io"},
		{"kreon-labs", "kreon-labs"},
		{"tweetmap", "tweetmap"},
		{"My.Project_2", "my-project-2"},
		{"  leading and trailing  ", "leading-and-trailing"},
		{"!!!", "site"},
		{"A__B---C..D", "a-b-c-d"},
		{"UPPERCASE", "uppercase"},
	}
	for _, c := range cases {
		if got := Slugify(c.in); got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestValidSlug(t *testing.T) {
	valid := []string{"sakkshm-kreon-labs-d24f", "my-site", "a", "abc123"}
	invalid := []string{"Weather.io", "-leading", "trailing-", "UPPER", "spaces here", "", "a..b", "a_b"}
	for _, s := range valid {
		if !ValidSlug(s) {
			t.Errorf("ValidSlug(%q) = false, want true", s)
		}
	}
	for _, s := range invalid {
		if ValidSlug(s) {
			t.Errorf("ValidSlug(%q) = true, want false", s)
		}
	}
}

func TestProjectSlug(t *testing.T) {
	got := ProjectSlug("sakkshm", "Weather.io", "5a05")
	want := "sakkshm-weather-io-5a05"
	if got != want {
		t.Errorf("ProjectSlug() = %q, want %q", got, want)
	}
	if len(ProjectSlug("sakkshm", "repo-with-a-very-long-name-that-goes-past-the-dns-label-limit", "abcd")) > MaxSlugLen {
		t.Error("ProjectSlug() exceeded MaxSlugLen")
	}
}
