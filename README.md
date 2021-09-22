# articles

Stuff I've written. All articles are licensed under a [Creative Commons Attribution-NoDerivatives 4.0 International License](https://creativecommons.org/licenses/by-nd/4.0/).

The top level is a Go program that generates HTML from the articles using [present](https://pkg.go.dev/golang.org/x/tools@v0.1.0/present). That program itself is licensed under a zlib license; the text thereof is at the top of `ssg.go`.

To add a new article, create a directory for it, then a `.article` file in that directory with the same name. E.g., an article called `madoka` needs `./madoka/madoka.article`. Supporting files (images, source code, &c.) can just be placed in the same directory, I think. Once the article is written, add it to `articles.json` under an appropriate section. Lastly, `go run . -out ../zephyrtronium.github.io -mf articles.json`.

Also don't forget to open a GitHub Discussion when publishing a new article.
