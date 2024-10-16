package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Member struct {
	Grade   string `json:"grade"`
	Name    string `json:"name"`
	NameEn  string `json:"name_en"`
	Bio     string `json:"bio"`
	ImgPath string `json:"img_path"`
}

func getMembers(c echo.Context) error {
	res, err := http.Get("https://omusp.jp/members")
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, struct{ Error string }{Error: "Failed to get members, cannot connect to omusp.jp"})
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Printf("status code error: %d %s", res.StatusCode, res.Status)
		return c.JSON(http.StatusInternalServerError, struct{ Error string }{Error: "Failed to get members, omusp.jp returned non-200 status code"})
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, struct{ Error string }{Error: "Failed to get members, cannot parse HTML from omusp.jp"})
	}

	members := []Member{}

	doc.Find(".user-card").Each(func(i int, s *goquery.Selection) {
		grade := s.Find(".user-pos").Text()
		name := s.Find(".user-name-ja").Text()
		name_en := s.Find(".user-name-eng").Text()
		bio := s.Find(".user-email").Text()
		image_path := strings.Replace(s.Find("img").AttrOr("src", ""), "https://omusp.jp", "", 1)
		member := Member{
			Grade:   grade,
			Name:    name,
			NameEn:  name_en,
			Bio:     bio,
			ImgPath: image_path,
		}
		members = append(members, member)
	})

	return c.JSON(http.StatusOK, struct {
		Members []Member `json:"members"`
	}{Members: members})
}

func main() {
	e := echo.New()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		HandleError: true, // forwards error to the global error handler, so it can decide appropriate status code
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				logger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
				)
			} else {
				logger.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("err", v.Error.Error()),
				)
			}
			return nil
		},
	}))

	e.GET("/", func(c echo.Context) error {
		return c.HTML(http.StatusOK, "Hello, Azure Container Apps!")
	})

	e.GET("/api/members", getMembers)

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, struct{ Status string }{Status: "OK"})
	})
	e.Logger.Fatal(e.Start(":8000"))
}
