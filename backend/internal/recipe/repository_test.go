package recipe

import "testing"

func TestResolveAutoParseImagesAppendsParsedImagesAfterExistingImages(t *testing.T) {
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
	if got, want := len(imageURLs), 3; got != want {
		t.Fatalf("len(imageURLs) = %d, want %d", got, want)
	}
	if got, want := imageURLs[2], "https://cdn.example.com/parsed-cover.jpg"; got != want {
		t.Fatalf("imageURLs[2] = %q, want %q", got, want)
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

func TestResolveAutoParseImagesDeduplicatesAndKeepsExistingOrder(t *testing.T) {
	t.Parallel()

	imageURL, imageURLs := resolveAutoParseImages(
		Recipe{
			ImageURL:  "https://cdn.example.com/manual-cover.jpg",
			ImageURLs: []string{"https://cdn.example.com/manual-cover.jpg", "https://cdn.example.com/shared.jpg"},
		},
		Recipe{
			ImageURL:  "https://cdn.example.com/shared.jpg",
			ImageURLs: []string{"https://cdn.example.com/shared.jpg", "https://cdn.example.com/parsed-extra.jpg"},
		},
	)

	if got, want := imageURL, "https://cdn.example.com/manual-cover.jpg"; got != want {
		t.Fatalf("imageURL = %q, want %q", got, want)
	}
	if got, want := len(imageURLs), 3; got != want {
		t.Fatalf("len(imageURLs) = %d, want %d", got, want)
	}
	if got, want := imageURLs[1], "https://cdn.example.com/shared.jpg"; got != want {
		t.Fatalf("imageURLs[1] = %q, want %q", got, want)
	}
	if got, want := imageURLs[2], "https://cdn.example.com/parsed-extra.jpg"; got != want {
		t.Fatalf("imageURLs[2] = %q, want %q", got, want)
	}
}

func TestNonNullableTrimmedStringPreservesEmptyString(t *testing.T) {
	t.Parallel()

	if got := nonNullableTrimmedString("   "); got != "" {
		t.Fatalf("nonNullableTrimmedString returned %q, want empty string", got)
	}
}

func TestNonNullableTrimmedStringTrimsWhitespace(t *testing.T) {
	t.Parallel()

	if got, want := nonNullableTrimmedString("  酸甜浓汤  "), "酸甜浓汤"; got != want {
		t.Fatalf("nonNullableTrimmedString = %q, want %q", got, want)
	}
}
