package victor

import (
    "github.com/brettbuddin/victor/campfire"
    "log"
    "time"
)

type Campfire struct {
    brain   *Brain
    account string
    token   string
    rooms   []int
    client  *campfire.Client
    me      *campfire.User
}

func NewCampfire(name string, account string, token string, rooms []int) *Campfire {
    return &Campfire{
        brain:   NewBrain(name),
        account: account,
        token:   token,
        rooms:   rooms,
        client:  campfire.NewClient(account, token),
    }
}

func (self *Campfire) Run() {
    log.Print("Starting up...")

    rooms := self.rooms

    channel := make(chan *campfire.Message)

    for i := range rooms {
        me, err := self.client.Me()

        if err != nil {
            log.Printf("Error fetching self: %s", err)
            continue
        }

        self.me = me

        go self.pollRoomDetails(rooms[i])

        room := self.client.Room(rooms[i])
        err = room.Join()

        if err != nil {
            log.Printf("Error joining room %i: %s", rooms[i], err)
            continue
        }

        room.Stream(channel)
        log.Print("Listening...")
    }

    for in := range channel {
        if in.UserId == self.me.Id {
            continue
        }

        if in.Type == "TextMessage" {
            msg := &TextMessage{
                Id:        in.Id,
                Body:      in.Body,
                CreatedAt: in.CreatedAt,

                Reply: self.reply(in.RoomId, in.UserId),
                Send: func(text string) {
                    self.client.Room(in.RoomId).Say(text)
                },
                Paste: func(text string) {
                    self.client.Room(in.RoomId).Paste(text)
                },
            }

            go self.brain.Receive(msg)
        }
    }
}

func (self *Campfire) Hear(expStr string, callback func(*TextMessage)) {
    self.brain.Hear(expStr, callback)
}

func (self *Campfire) Respond(expStr string, callback func(*TextMessage)) {
    self.brain.Respond(expStr, callback)
}

func (self *Campfire) pollRoomDetails(roomId int) {
    for {
        details, err := self.client.Room(roomId).Show()

        if err != nil {
            continue
        }

        for _, user := range details.Users {
            self.brain.RememberUser(&User{Id: user.Id, Name: user.Name})
        }

        time.Sleep(300 * time.Second)
    }
}

func (self *Campfire) reply(roomId int, userId int) func(string) {
    room := self.client.Room(roomId)
    user := self.brain.UserForId(userId)
    prefix := ""

    if user != nil {
        prefix = user.Name + ": "
    }

    return func(text string) {
        room.Say(prefix + text)
    }
}
