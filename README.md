# spotify-playlister

get a list of all songs & their artists within a playlist.

## How to use

1. make an app on spotify dev. dashboard
    * set the redir. URI to `http://localhost:8080/callback`
    * select `Web API` under APIs Used
2. put the ClientID & Secret into a `.env` file at project root
    * alternatively export or source the envs
```bash
  cat > .env << EOF
  SPOTIFY_ID="idgoeshere"
  SPOTIFY_SECRET="wowsosecret"
  EOF
```
4. run the program normally
     * if running using an executable, the ID & Secrets will need to be made available to the program

---
> Suggestions are welcome.
