package ytvideodata

import (
	"fmt"
	"net/http"

	"golang.org/x/net/html"
)

func getFromPage(videoId string) (*VideoData, error) {
	resp, err := http.Get("https://youtu.be/" + videoId)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	var videoData VideoData
	videoData.Title = getTitle(doc)
	videoData.ThumbnailUrl = fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoId)
	videoData.AuthorName = getLinkContent(doc)
	return &videoData, nil
}

func getTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		return n.FirstChild.Data
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := getTitle(c); title != "" {
			return title
		}
	}
	return ""
}

func getLinkContent(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "link" {
		for _, attr := range n.Attr {
			if attr.Key == "itemprop" && attr.Val == "name" {
				for _, attr := range n.Attr {
					if attr.Key == "content" {
						return attr.Val
					}
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if content := getLinkContent(c); content != "" {
			return content
		}
	}
	return ""
}
