package trakt

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xanderstrike/plexhooks"
	"testing"
)

type MockTraktClient struct {
	mock.Mock
	MakeRequestResponses map[string][]byte
}

func (m *MockTraktClient) MakeRequest(url string) []byte {
	args := m.Called(url)

	if resp, ok := m.MakeRequestResponses[url]; ok {
		return resp
	}

	return args.Get(0).([]byte)
}

func (m *MockTraktClient) ScrobbleRequest(action string, body []byte, token string) []byte {
	args := m.Called(action, body, token)

	return args.Get(0).([]byte)
}

func TestFindEpisode(t *testing.T) {

	type testCase struct {
		Name              string
		Payload           plexhooks.PlexResponse
		ApiCallsResponses map[string][]byte
		ExpectedEpisode   Episode
	}

	var cases = []testCase{
		{
			Name: "Severance S02E03 with tvdb",
			Payload: plexhooks.PlexResponse{
				Metadata: plexhooks.Metadata{
					Guid: "plex://episode/669013e833d03eeac35f5d09",
					ExternalGuid: []plexhooks.ExternalGuid{
						{
							Id: "tvdb://10592760",
						},
						{
							Id: "tmdb://5469117",
						},
						{
							Id: "imdb://tt15241840",
						},
					},
				},
			},
			ApiCallsResponses: map[string][]byte{
				"https://api.trakt.tv/search/tvdb/10592760?type=episode": []byte(`[
            {"episode": {
				"title": "Who Is Alive?",
				"season": 2, 
				"number": 3,
				"ids": {"trakt": 12103029, "tvdb": 10592760, "imdb": "tt15241840", "tmdb": 5469117, "tvrage": 0}
			}}
        ]`),
			},
			ExpectedEpisode: Episode{
				Title:  "Who Is Alive?",
				Season: 2,
				Number: 3,
				Ids: Ids{
					Trakt:  12103029,
					Tvdb:   10592760,
					Imdb:   "tt15241840",
					Tmdb:   5469117,
					Tvrage: 0,
				},
			},
		},
		{
			Name: "Severance S02E03 with tmdb",
			Payload: plexhooks.PlexResponse{
				Metadata: plexhooks.Metadata{
					Guid: "plex://episode/669013e833d03eeac35f5d09",
					ExternalGuid: []plexhooks.ExternalGuid{
						{
							Id: "tmdb://5469117",
						},
						{
							Id: "imdb://tt15241840",
						},
					},
				},
			},
			ApiCallsResponses: map[string][]byte{
				"https://api.trakt.tv/search/tmdb/5469117?type=episode": []byte(`[
            {"episode": {
				"title": "Who Is Alive?",
				"season": 2, 
				"number": 3,
				"ids": {"trakt": 12103029, "tvdb": 10592760, "imdb": "tt15241840", "tmdb": 5469117, "tvrage": 0}
			}}
        ]`),
			},
			ExpectedEpisode: Episode{
				Title:  "Who Is Alive?",
				Season: 2,
				Number: 3,
				Ids: Ids{
					Trakt:  12103029,
					Tvdb:   10592760,
					Imdb:   "tt15241840",
					Tmdb:   5469117,
					Tvrage: 0,
				},
			},
		},
		{
			Name: "Severance S02E03 with imdb",
			Payload: plexhooks.PlexResponse{
				Metadata: plexhooks.Metadata{
					Guid: "plex://episode/669013e833d03eeac35f5d09",
					ExternalGuid: []plexhooks.ExternalGuid{
						{
							Id: "imdb://tt15241840",
						},
					},
				},
			},
			ApiCallsResponses: map[string][]byte{
				"https://api.trakt.tv/search/imdb/tt15241840?type=episode": []byte(`[
            {"episode": {
				"title": "Who Is Alive?",
				"season": 2, 
				"number": 3,
				"ids": {"trakt": 12103029, "tvdb": 10592760, "imdb": "tt15241840", "tmdb": 5469117, "tvrage": 0}
			}}
        ]`),
			},
			ExpectedEpisode: Episode{
				Title:  "Who Is Alive?",
				Season: 2,
				Number: 3,
				Ids: Ids{
					Trakt:  12103029,
					Tvdb:   10592760,
					Imdb:   "tt15241840",
					Tmdb:   5469117,
					Tvrage: 0,
				},
			},
		},
		{
			Name: "Severance S02E03 with title",
			Payload: plexhooks.PlexResponse{
				Metadata: plexhooks.Metadata{
					Guid:             "plex://episode/669013e833d03eeac35f5d09",
					GrandparentTitle: "Severance",
					Index:            3,
					ParentIndex:      2,
					Year:             2022,
				},
			},
			ApiCallsResponses: map[string][]byte{
				"https://api.trakt.tv/search/show?query=Severance": []byte(`[
            {"type":"show","score":97.20612,"show":{"title":"Severance","year":2022,"ids":{"trakt":154997,"slug":"severance","tvdb":371980,"imdb":"tt11280740","tmdb":95396,"tvrage":null}}}
        ]`),
				"https://api.trakt.tv/shows/154997/seasons?extended=episodes": []byte(`
[{"number":0,"ids":{"trakt":444956,"tvdb":1985531,"tmdb":443382,"tvrage":null},"episodes":[{"season":0,"number":1,"title":"Welcome to Lumon","ids":{"trakt":12840119,"tvdb":10948626,"imdb":"tt26448621","tmdb":5995805,"tvrage":null}}]},{"number":1,"ids":{"trakt":236735,"tvdb":1971742,"tmdb":135726,"tvrage":null},"episodes":[{"season":1,"number":1,"title":"Good News About Hell","ids":{"trakt":4648809,"tvdb":8891221,"imdb":"tt11650328","tmdb":1982925,"tvrage":null}},{"season":1,"number":2,"title":"Half Loop","ids":{"trakt":5769062,"tvdb":8891222,"imdb":"tt13393872","tmdb":3396429,"tvrage":null}},{"season":1,"number":3,"title":"In Perpetuity","ids":{"trakt":5769063,"tvdb":8891223,"imdb":"tt13399816","tmdb":3396430,"tvrage":null}},{"season":1,"number":4,"title":"The You You Are","ids":{"trakt":5769064,"tvdb":8891224,"imdb":"tt13411248","tmdb":3396431,"tvrage":null}},{"season":1,"number":5,"title":"The Grim Barbarity of Optics and Design","ids":{"trakt":5769066,"tvdb":8891225,"imdb":"tt13424090","tmdb":3396432,"tvrage":null}},{"season":1,"number":6,"title":"Hide and Seek","ids":{"trakt":5769067,"tvdb":8891226,"imdb":"tt13424092","tmdb":3396433,"tvrage":null}},{"season":1,"number":7,"title":"Defiant Jazz","ids":{"trakt":5769068,"tvdb":8891227,"imdb":"tt13424094","tmdb":3396434,"tvrage":null}},{"season":1,"number":8,"title":"What's for Dinner?","ids":{"trakt":5769069,"tvdb":8891228,"imdb":"tt13424096","tmdb":3396435,"tvrage":null}},{"season":1,"number":9,"title":"The We We Are","ids":{"trakt":5769070,"tvdb":8891229,"imdb":"tt13424098","tmdb":3396436,"tvrage":null}}]},{"number":2,"ids":{"trakt":324357,"tvdb":2136511,"tmdb":401674,"tvrage":null},"episodes":[{"season":2,"number":1,"title":"Hello, Ms. Cobel","ids":{"trakt":12102389,"tvdb":10586352,"imdb":"tt15180436","tmdb":5469028,"tvrage":null}},{"season":2,"number":2,"title":"Goodbye, Mrs. Selvig","ids":{"trakt":12103028,"tvdb":10592759,"imdb":"tt15237910","tmdb":5469115,"tvrage":null}},{"season":2,"number":3,"title":"Who Is Alive?","ids":{"trakt":12103029,"tvdb":10592760,"imdb":"tt15241840","tmdb":5469117,"tvrage":null}},{"season":2,"number":4,"title":"Woe's Hollow","ids":{"trakt":12103030,"tvdb":10592761,"imdb":"tt15241844","tmdb":5469118,"tvrage":null}},{"season":2,"number":5,"title":"Trojan's Horse","ids":{"trakt":12103031,"tvdb":10592762,"imdb":"tt15242966","tmdb":5469120,"tvrage":null}},{"season":2,"number":6,"title":"Attila","ids":{"trakt":12103032,"tvdb":10592763,"imdb":"tt15242972","tmdb":5469123,"tvrage":null}},{"season":2,"number":7,"title":"Chikhai Bardo","ids":{"trakt":12103033,"tvdb":10592764,"imdb":"tt15242980","tmdb":5469128,"tvrage":null}},{"season":2,"number":8,"title":"Sweet Vitriol","ids":{"trakt":12103034,"tvdb":10592765,"imdb":"tt15242986","tmdb":5469133,"tvrage":null}},{"season":2,"number":9,"title":"The After Hours","ids":{"trakt":12103035,"tvdb":10592766,"imdb":"tt15242994","tmdb":5469139,"tvrage":null}},{"season":2,"number":10,"title":"Cold Harbor","ids":{"trakt":12103036,"tvdb":10592768,"imdb":"tt15242998","tmdb":5469142,"tvrage":null}}]},{"number":3,"ids":{"trakt":453841,"tvdb":null,"tmdb":447475,"tvrage":null},"episodes":[]}]
`),
			},
			ExpectedEpisode: Episode{
				Title:  "Who Is Alive?",
				Season: 2,
				Number: 3,
				Ids: Ids{
					Trakt:  12103029,
					Tvdb:   10592760,
					Imdb:   "tt15241840",
					Tmdb:   5469117,
					Tvrage: 0,
				},
			},
		},
		{
			Name: "37 secondes S01E05 with title",
			Payload: plexhooks.PlexResponse{
				Metadata: plexhooks.Metadata{
					Guid:             "plex://episode/669013e833d03eeac35f5d09",
					GrandparentTitle: "37 secondes",
					Index:            5,
					ParentIndex:      1,
					Year:             2025,
					Title:            "",
				},
			},
			ApiCallsResponses: map[string][]byte{
				"https://api.trakt.tv/search/show?query=37%20secondes": []byte(`[
            {"type":"show","score":119.65823,"show":{"title":"37 secondes","year":2025,"ids":{"trakt":232082,"slug":"37-secondes","tvdb":460657,"imdb":"tt32228145","tmdb":237811,"tvrage":null}}}
        ]`),
				"https://api.trakt.tv/shows/232082/seasons?extended=episodes": []byte(`

[{"number":1,"ids":{"trakt":374539,"tvdb":null,"tmdb":362030,"tvrage":null},"episodes":[{"season":1,"number":1,"title":"Episode 1","ids":{"trakt":11658360,"tvdb":null,"imdb":null,"tmdb":4837828,"tvrage":null}},{"season":1,"number":2,"title":"Episode 2","ids":{"trakt":11722088,"tvdb":null,"imdb":null,"tmdb":5322127,"tvrage":null}},{"season":1,"number":3,"title":"Episode 3","ids":{"trakt":11722089,"tvdb":null,"imdb":null,"tmdb":5322128,"tvrage":null}},{"season":1,"number":4,"title":"Episode 4","ids":{"trakt":11722090,"tvdb":null,"imdb":null,"tmdb":5322129,"tvrage":null}},{"season":1,"number":5,"title":"Episode 5","ids":{"trakt":11722092,"tvdb":null,"imdb":null,"tmdb":5322130,"tvrage":null}},{"season":1,"number":6,"title":"Episode 6","ids":{"trakt":11722093,"tvdb":null,"imdb":null,"tmdb":5322131,"tvrage":null}}]}]`),
			},
			ExpectedEpisode: Episode{
				Title:  "Episode 5",
				Season: 1,
				Number: 5,
				Ids: Ids{
					Trakt:  11722092,
					Tvdb:   0,
					Imdb:   "",
					Tmdb:   5322130,
					Tvrage: 0,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			client := &MockTraktClient{
				MakeRequestResponses: c.ApiCallsResponses,
			}
			client.On("MakeRequest", mock.Anything)

			episode := findEpisode(client, c.Payload)

			assert.Equal(t, c.ExpectedEpisode.Title, episode.Title, "Title mismatch")
			assert.Equal(t, c.ExpectedEpisode.Season, episode.Season, "Season mismatch")
			assert.Equal(t, c.ExpectedEpisode.Number, episode.Number, "Number mismatch")
			assert.Equal(t, c.ExpectedEpisode.Ids, episode.Ids, "Ids mismatch")
		})
	}
}

func TestFindMovie(t *testing.T) {

	type testCase struct {
		Name              string
		Payload           plexhooks.PlexResponse
		ApiCallsResponses map[string][]byte
		ExpectedMovie     Movie
	}

	var cases = []testCase{
		{
			Name: "Apollo 13 with tmdb",
			Payload: plexhooks.PlexResponse{
				Metadata: plexhooks.Metadata{
					Guid: "plex://movie/5d776826151a60001f24a9d0",
					ExternalGuid: []plexhooks.ExternalGuid{
						{
							Id: "tmdb://568",
						},
						{
							Id: "imdb://tt0112384",
						},
					},
				},
			},
			ApiCallsResponses: map[string][]byte{
				"https://api.trakt.tv/search/tmdb/568?type=movie": []byte(`
[{"type":"movie","score":1000,"movie":{"title":"Apollo 13","year":1995,"ids":{"trakt":448,"slug":"apollo-13-1995","imdb":"tt0112384","tmdb":568}}}]`)},
			ExpectedMovie: Movie{
				Title: "Apollo 13",
				Year:  1995,
				Ids: Ids{
					Trakt:  448,
					Tvdb:   0,
					Imdb:   "tt0112384",
					Tmdb:   568,
					Tvrage: 0,
				},
			},
		},
		{
			Name: "Apollo 13 with imdb",
			Payload: plexhooks.PlexResponse{
				Metadata: plexhooks.Metadata{
					Guid: "plex://movie/5d776826151a60001f24a9d0",
					ExternalGuid: []plexhooks.ExternalGuid{
						{
							Id: "imdb://tt0112384",
						},
					},
				},
			},
			ApiCallsResponses: map[string][]byte{
				"https://api.trakt.tv/search/imdb/tt0112384?type=movie": []byte(`
[{"type":"movie","score":1000,"movie":{"title":"Apollo 13","year":1995,"ids":{"trakt":448,"slug":"apollo-13-1995","imdb":"tt0112384","tmdb":568}}}]`)},
			ExpectedMovie: Movie{
				Title: "Apollo 13",
				Year:  1995,
				Ids: Ids{
					Trakt:  448,
					Tvdb:   0,
					Imdb:   "tt0112384",
					Tmdb:   568,
					Tvrage: 0,
				},
			},
		},
		{
			Name: "Apollo 13 with title",
			Payload: plexhooks.PlexResponse{
				Metadata: plexhooks.Metadata{
					Guid:  "plex://movie/5d776826151a60001f24a9d0",
					Title: "Apollo 13",
					Year:  1995,
				},
			},
			ApiCallsResponses: map[string][]byte{
				"https://api.trakt.tv/search/movie?query=Apollo%2013": []byte(`
[{"type":"movie","score":109.99295,"movie":{"title":"Apollo 13","year":1995,"ids":{"trakt":448,"slug":"apollo-13-1995","imdb":"tt0112384","tmdb":568}}},{"type":"movie","score":1937.7615,"movie":{"title":"Apollo 13: Survival","year":2024,"ids":{"trakt":1012041,"slug":"apollo-13-survival-2024","imdb":"tt31852716","tmdb":1249216}}},{"type":"movie","score":1390.2266,"movie":{"title":"13 Factors That Saved Apollo 13","year":2014,"ids":{"trakt":306899,"slug":"13-factors-that-saved-apollo-13-2014","imdb":"tt3884428","tmdb":363864}}},{"type":"movie","score":1286.7507,"movie":{"title":"Apollo 13: Home Safe","year":2020,"ids":{"trakt":537216,"slug":"apollo-13-home-safe-2020","imdb":null,"tmdb":692970}}},{"type":"movie","score":1281.8206,"movie":{"title":"Lost Moon: The Triumph of Apollo 13","year":1996,"ids":{"trakt":94420,"slug":"lost-moon-the-triumph-of-apollo-13-1996","imdb":"tt0327018","tmdb":141498}}},{"type":"movie","score":1186.8324,"movie":{"title":"Apollo 13: The Untold Story","year":1995,"ids":{"trakt":173213,"slug":"apollo-13-the-untold-story-1995","imdb":null,"tmdb":275435}}},{"type":"movie","score":1155.6042,"movie":{"title":"Apollo 13: To the Edge and Back","year":1994,"ids":{"trakt":88112,"slug":"apollo-13-to-the-edge-and-back-1994","imdb":"tt0180443","tmdb":128857}}},{"type":"movie","score":1106.6807,"movie":{"title":"Salyut-7","year":2017,"ids":{"trakt":284879,"slug":"salyut-7-2017","imdb":"tt6537238","tmdb":438740}}},{"type":"movie","score":1087.2601,"movie":{"title":"Apollo 11","year":2019,"ids":{"trakt":399358,"slug":"apollo-11-2019","imdb":"tt8760684","tmdb":549559}}},{"type":"movie","score":1085.604,"movie":{"title":"Apartment 1303 3D","year":2012,"ids":{"trakt":103563,"slug":"apartment-1303-3d-2012","imdb":"tt1540767","tmdb":160070}}}]`),
			},
			ExpectedMovie: Movie{
				Title: "Apollo 13",
				Year:  1995,
				Ids: Ids{
					Trakt:  448,
					Tvdb:   0,
					Imdb:   "tt0112384",
					Tmdb:   568,
					Tvrage: 0,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			client := &MockTraktClient{
				MakeRequestResponses: c.ApiCallsResponses,
			}
			client.On("MakeRequest", mock.Anything)

			movie := findMovie(client, c.Payload)
			assert.Equal(t, c.ExpectedMovie.Title, movie.Title, "Title mismatch")
			assert.Equal(t, c.ExpectedMovie.Year, movie.Year, "Year mismatch")
			assert.Equal(t, c.ExpectedMovie.Ids, movie.Ids, "Ids mismatch")
		})
	}
}
