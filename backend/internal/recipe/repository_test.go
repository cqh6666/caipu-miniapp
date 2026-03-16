package recipe

import "testing"

func TestResolveAutoParseImagesKeepsExistingManualImages(t *testing.T) {
	t.Parallel()

	current := Recipe{
		ImageURL:  "https://cdn.example.com/manual-cover.jpg",
		ImageURLs: []string{"https://cdn.example.com/manual-cover.jpg", "https://cdn.example.com/manual-step.jpg"},
	}
	draft := Recipe{
		ImageURL:  "https://cdn.example.com/parsed-cover.jpg",
		ImageURLs: []string{"https://cdn.example.com/parsed-cover.jpg"},
	}

	imageURL, imageURLs := resolveAutoParseImages(current, draft)
	if got, want := imageURL, "https://cdn.example.com/manual-cover.jpg"; got != want {
		t.Fatalf("imageURL = %q, want %q", got, want)
	}
	if got, want := len(imageURLs), 2; got != want {
		t.Fatalf("len(imageURLs) = %d, want %d", got, want)
	}
}

func TestResolveAutoParseImagesBackfillsWhenRecipeHasNoImages(t *testing.T) {
	t.Parallel()

	imageURL, imageURLs := resolveAutoParseImages(Recipe{}, Recipe{
		ImageURL:  "https://cdn.example.com/parsed-cover.jpg",
		ImageURLs: []string{"https://cdn.example.com/parsed-cover.jpg", "https://cdn.example.com/parsed-step.jpg"},
	})

	if got, want := imageURL, "https://cdn.example.com/parsed-cover.jpg"; got != want {
		t.Fatalf("imageURL = %q, want %q", got, want)
	}
	if got, want := len(imageURLs), 2; got != want {
		t.Fatalf("len(imageURLs) = %d, want %d", got, want)
	}
}
