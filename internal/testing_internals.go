package internal

import (
	"fmt"
	"log"
	"time"

	"github.com/2beens/serjtubincom/internal/aerospike"
	"github.com/2beens/serjtubincom/internal/blog"
	"github.com/2beens/serjtubincom/internal/cache"
)

const (
	blogPostsCount = 5
)

type testingInternals struct {
	// board
	aeroTestClient       *aerospike.BoardAeroTestClient
	board                *Board
	boardCache           *cache.BoardTestCache
	initialBoardMessages map[int]*BoardMessage
	lastInitialMessage   *BoardMessage

	// blog
	blogApi      *blog.TestApi
	loginSession *LoginSession
}

func newTestingInternals() *testingInternals {
	now := time.Now()
	initialBoardMessages := map[int]*BoardMessage{
		0: {
			ID:        0,
			Author:    "serj",
			Timestamp: now.Add(-time.Hour).Unix(),
			Message:   "test message blabla",
		},
		1: {
			ID:        1,
			Author:    "serj",
			Timestamp: now.Unix(),
			Message:   "test message gragra",
		},
		2: {
			ID:        2,
			Author:    "ana",
			Timestamp: now.Add(-2 * time.Hour).Unix(),
			Message:   "test message aaaaa",
		},
		3: {
			ID:        3,
			Author:    "drago",
			Timestamp: now.Add(-5 * 24 * time.Hour).Unix(),
			Message:   "drago's test message aaaaa sve",
		},
		4: {
			ID:        4,
			Author:    "rodjak nenad",
			Timestamp: now.Add(-2 * time.Minute).Unix(),
			Message:   "ja se mislim sta'e bilo",
		},
	}

	aeroClient := aerospike.NewBoardAeroTestClient()
	boardCache := cache.NewBoardTestCache()

	board, err := NewBoard(aeroClient, boardCache)
	if err != nil {
		log.Fatal(err)
	}

	if err := board.StoreMessage(*initialBoardMessages[0]); err != nil {
		panic(err)
	}
	if err := board.StoreMessage(*initialBoardMessages[1]); err != nil {
		panic(err)
	}
	if err := board.StoreMessage(*initialBoardMessages[2]); err != nil {
		panic(err)
	}
	if err := board.StoreMessage(*initialBoardMessages[3]); err != nil {
		panic(err)
	}
	if err := board.StoreMessage(*initialBoardMessages[4]); err != nil {
		panic(err)
	}

	// FIXME: when storing messages in a loop, we got some race condition
	// indication of a design smell
	//for _, m := range initialBoardMessages {
	//	fmt.Printf("++ %d %s: %d\n", m.ID, m.Author, m.Timestamp)
	//	if err := board.StoreMessage(*m); err != nil {
	//		panic(err)
	//	}
	//}

	boardCache.ClearFunctionCallsLog()

	// blog stuff
	blogApi := blog.NewBlogTestApi()
	for i := 0; i < blogPostsCount; i++ {
		if err = blogApi.AddBlog(&blog.Blog{
			Id:        i,
			Title:     fmt.Sprintf("blog%dtitle", i),
			CreatedAt: now,
			Content:   fmt.Sprintf("blog %d content", i),
		}); err != nil {
			panic(err)
		}
	}

	loginSession := &LoginSession{
		Token:     "tokenAbc123",
		CreatedAt: now,
		TTL:       0,
	}

	return &testingInternals{
		aeroTestClient:       aeroClient,
		board:                board,
		boardCache:           boardCache,
		initialBoardMessages: initialBoardMessages,
		lastInitialMessage:   initialBoardMessages[1],
		blogApi:              blogApi,
		loginSession:         loginSession,
	}
}