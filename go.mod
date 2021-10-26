module diskoi

go 1.17

require (
	github.com/bwmarrin/discordgo v0.23.3-0.20211010150959-f0b7e81468f7
	github.com/davecgh/go-spew v1.1.1
	github.com/stretchr/testify v1.7.0
)

require (
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b // indirect
	golang.org/x/sys v0.0.0-20201119102817-f84b799fce68 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

replace github.com/bwmarrin/discordgo => ./local/discordgo
