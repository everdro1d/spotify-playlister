package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/browser"

	"github.com/zmb3/spotify/v2"
	"github.com/zmb3/spotify/v2/auth"
)

import _ "github.com/joho/godotenv/autoload"

const redirectURI = "http://localhost:8080/callback"

var (
    auth = spotifyauth.New(
        spotifyauth.WithRedirectURL(redirectURI),
        spotifyauth.WithScopes(
            spotifyauth.ScopePlaylistReadPrivate,
            spotifyauth.ScopePlaylistReadCollaborative,
        ),
    )
    ch = make(chan *spotify.Client)
    state = "wowastate"
    ctx = context.Background()
)

func check(err error) {
    if err != nil {
        log.Fatal(err)
    }
}

func main() {
    startLogFile()

    http.HandleFunc("/callback", completeAuth)
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        log.Println("Got request for:", r.URL.String())
    })

    go func() {
        err := http.ListenAndServe(":8080", nil)
        check(err)
    }()

    url := auth.AuthURL(state)
    fmt.Println("Please login to Spotify with the following link:", url)
    browser.OpenURL(url)

    client := <-ch

    user, err := client.CurrentUser(ctx)
    check(err)
    fmt.Println("Logged in as:", user.ID)

    // ask user to input link to playlist
    // read from 'playlist/' to '?' or until EOL, whichever comes first
    playlistID := ""
    for playlistID == "" {
        fmt.Print("Enter playlist share URL: ")
        var playlistShareURL string
        fmt.Scanln(&playlistShareURL)

        playlistID = getStringBetween(playlistShareURL, "playlist/", "?")

        if playlistID == "" {
            fmt.Println("Please check the link and try again.")
        }
    }

    // get playlist stuff
    getPlaylistInfo(playlistID, *client)
}

func startLogFile() {
    f, err := os.Create("./latest.log")
    check(err)
    defer f.Close()

    log.SetOutput(f)
    log.Println("BGN ERR LOG")
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
    token, err := auth.Token(r.Context(), state, r)
    if err != nil {
        http.Error(w, "Couldn't get token.", http.StatusForbidden)
        log.Fatal(err)
    }
    if st := r.FormValue("state"); st != state {
        http.NotFound(w, r)
        log.Fatalf("State mismatch: %s != %s/n", st, state)
    }

    client := spotify.New(auth.Client(r.Context(), token))
    fmt.Fprintf(w, "Login Success. Please return to program.")
    ch <- client
}

func getStringBetween(str string, start string, end string) (result string) {
    s := strings.Index(str, start)
    if s == -1 {
        return ""
    }
    s += len(start)
    e := strings.Index(str[s:], end)
    if e == -1 {
        e = len(str[s:])
    }

    return str[s : s+e]
}

func getPlaylistInfo(id string, client spotify.Client) {
    playlist, err := client.GetPlaylist(
        ctx,
        spotify.ID(id),
    )
    check(err)

    tracks, err := client.GetPlaylistItems(
        ctx,
        spotify.ID(id),
    )
    check(err)

    fmt.Println("Please Select Output Type:")
    fmt.Println("1. Console\n2. File")
    var console bool
    for finish := false; !finish; {
        fmt.Print("Select Number: ")
        var input string
        fmt.Scanln(&input)
        if strings.Contains(input, "1") {
            console, finish = true, true
        } else if strings.Contains(input, "2") {
            console, finish = false, true
        } else {
            fmt.Println("Invalid Option. Please try again.")
        }
    }

    // save info somehow
    exportPlaylistInfo(id, client, tracks, playlist, err, console)
}

func exportPlaylistInfo(
    id string,
    client spotify.Client,
    tracks *spotify.PlaylistItemPage,
    playlist *spotify.FullPlaylist,
    err error,
    console bool,
) {
    ioWriter := os.Stdout
    // switch between console output and file
    if console != true {
        file, fileErr := os.Create("./" + playlist.Name + ".txt")
        check(fileErr)
        defer file.Close()
        ioWriter = file
        fmt.Printf("Saved to file: \"%s\"\n", file.Name())
    }
    w := bufio.NewWriter(ioWriter)

    fmt.Fprintln(w, "+")
    fmt.Fprintf(w, "| Playlist Name: \"%s\"\n", playlist.Name)
    fmt.Fprintf(w, "| Owner: \"%s\"\n", playlist.Owner.DisplayName)
    fmt.Fprintf(w, "| Open Playlist: \"%s\"\n", "https://open.spotify.com/playlist/" + id)
    fmt.Fprintln(w, "+")
    fmt.Fprintf(w, "| Playlist has %d total tracks.\n", tracks.Total)
    // page
    for page := 1; ; page++ {
        fmt.Fprintf(w, "| Page %d has %d tracks.\n", page, len(tracks.Items))
        fmt.Fprintln(w, "+")

        // track
        for i := 0; i < len(tracks.Items); i++ {
            name := tracks.Items[i].Track.Track.Name
            artists := ""
            artistsArr := tracks.Items[i].Track.Track.Artists
            for i := 0; i < len(artistsArr); i++ {
                inter := ""
                if artists != "" {
                    inter = ", "
                }
                artists += inter + artistsArr[i].Name
            }
            fmt.Fprintf(
                w,
                "%d)\n| Name: \"%-s\" \n| > Artist(s): \"%-s\" \n",
                i+1,
                name,
                artists,
            )
        }
        fmt.Fprintln(w, "+")

        err = client.NextPage(ctx, tracks)
        if err == spotify.ErrNoMorePages {
            break
        }
        check(err)
    }

    w.Flush()
}
