package nyb

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/ugjka/dumbirc"
	"github.com/ugjka/go-tz"
)

var nickChangeInterval = time.Second * 5

func (bot *Settings) addCallbacks() {
	irc := bot.IrcConn
	//On any message send a signal to ping timer to be ready
	irc.AddCallback(dumbirc.ANYMESSAGE, func(msg *dumbirc.Message) {
		pingpong(bot.pp)
	})

	irc.AddCallback(dumbirc.WELCOME, func(msg *dumbirc.Message) {
		irc.Join(bot.IrcChans)
		//Prevent early start
		bot.Do(func() {
			close(bot.start)
		})
	})

	irc.AddCallback(dumbirc.PING, func(msg *dumbirc.Message) {
		log.Println("PING recieved, sending PONG")
		irc.Pong()
	})

	irc.AddCallback(dumbirc.PONG, func(msg *dumbirc.Message) {
		log.Println("Got PONG...")
	})

	irc.AddCallback(dumbirc.NICKTAKEN, func(msg *dumbirc.Message) {
		log.Println("Nick taken, changing...")
		time.Sleep(nickChangeInterval)
		irc.Nick = changeNick(irc.Nick)
		log.Printf("New nick: %s", irc.Nick)
		irc.NewNick(irc.Nick)
	})
}

func (bot *Settings) addTriggers() {
	irc := bot.IrcConn
	//Trigger for !help
	stHelp := "%s: Query location: '%s <location>', Next zone: '%s !next', Last zone: '%s !last', Remaining: '%s !remaining', Source code: https://github.com/ugjka/newyearsbot"
	irc.AddTrigger(dumbirc.Trigger{
		Condition: func(msg *dumbirc.Message) bool {
			return msg.Command == dumbirc.PRIVMSG &&
				msg.Trailing == fmt.Sprintf("%s !help", bot.IrcTrigger)
		},
		Response: func(msg *dumbirc.Message) {
			irc.Reply(msg, fmt.Sprintf(stHelp, msg.Name, bot.IrcTrigger, bot.IrcTrigger, bot.IrcTrigger, bot.IrcTrigger))
		},
	})
	//Trigger for !next
	irc.AddTrigger(dumbirc.Trigger{
		Condition: func(msg *dumbirc.Message) bool {
			return msg.Command == dumbirc.PRIVMSG &&
				msg.Trailing == fmt.Sprintf("%s !next", bot.IrcTrigger)
		},
		Response: func(msg *dumbirc.Message) {
			log.Println("Querying !next...")
			dur := time.Minute * time.Duration(bot.next.Offset*60)
			if timeNow().UTC().Add(dur).After(target) {
				irc.Reply(msg, fmt.Sprintf("No more next, %d is here AoE", target.Year()))
				return
			}
			humandur := durafmt.Parse(target.Sub(timeNow().UTC().Add(dur)))
			irc.Reply(msg, fmt.Sprintf("Next New Year in %s in %s",
				removeMilliseconds(humandur), bot.next))
		},
	})
	//Trigger for !last
	irc.AddTrigger(dumbirc.Trigger{
		Condition: func(msg *dumbirc.Message) bool {
			return msg.Command == dumbirc.PRIVMSG &&
				msg.Trailing == fmt.Sprintf("%s !last", bot.IrcTrigger)
		},
		Response: func(msg *dumbirc.Message) {
			log.Println("Querying !last...")
			dur := time.Minute * time.Duration(bot.last.Offset*60)
			humandur := durafmt.Parse(timeNow().UTC().Add(dur).Sub(target))
			if bot.last.Offset == -12 {
				humandur = durafmt.Parse(timeNow().UTC().Add(dur).Sub(target.AddDate(-1, 0, 0)))
			}
			irc.Reply(msg, fmt.Sprintf("Last New Year %s ago in %s",
				removeMilliseconds(humandur), bot.last))
		},
	})
	//Trigger for !remaining
	irc.AddTrigger(dumbirc.Trigger{
		Condition: func(msg *dumbirc.Message) bool {
			return msg.Command == dumbirc.PRIVMSG &&
				msg.Trailing == fmt.Sprintf("%s !remaining", bot.IrcTrigger)
		},
		Response: func(msg *dumbirc.Message) {
			ss := "s"
			if bot.remaining == 1 {
				ss = ""
			}
			irc.Reply(msg, fmt.Sprintf("%s: %d timezone%s remaining", msg.Name, bot.remaining, ss))
		},
	})
	//Trigger for location queries
	irc.AddTrigger(dumbirc.Trigger{
		Condition: func(msg *dumbirc.Message) bool {
			return msg.Command == dumbirc.PRIVMSG &&
				!strings.Contains(msg.Trailing, "!next") &&
				!strings.Contains(msg.Trailing, "!last") &&
				!strings.Contains(msg.Trailing, "!help") &&
				!strings.Contains(msg.Trailing, "!remaining") &&
				strings.HasPrefix(msg.Trailing, fmt.Sprintf("%s ", bot.IrcTrigger))
		},
		Response: func(msg *dumbirc.Message) {
			tz, err := bot.getNewYear(msg.Trailing[len(bot.IrcTrigger)+1:])
			if err == errNoZone || err == errNoPlace {
				log.Println("Query error:", err)
				irc.Reply(msg, fmt.Sprintf("%s: %s", msg.Name, err))
				return
			}
			if err != nil {
				log.Println("Query error:", err)
				irc.Reply(msg, fmt.Sprintf("%s: Some error occurred!", msg.Name))
				return
			}
			irc.Reply(msg, fmt.Sprintf("%s: %s", msg.Name, tz))
		},
	})
}

var (
	errNoZone  = errors.New("couldn't get the timezone for that location")
	errNoPlace = errors.New("Couldn't find that place")
)

func (bot *Settings) getNominatimReqURL(location *string) string {
	maps := url.Values{}
	maps.Add("q", *location)
	maps.Add("format", "json")
	maps.Add("accept-language", "en")
	maps.Add("limit", "1")
	maps.Add("email", bot.Email)
	return bot.Nominatim + NominatimEndpoint + maps.Encode()
}

var stNewYearWillHappen = "New Year in %s will happen in %s"
var stNewYearHappenned = "New Year in %s happened %s ago"

func (bot *Settings) getNewYear(location string) (string, error) {
	log.Println("Querying location:", location)
	data, err := NominatimGetter(bot.getNominatimReqURL(&location))
	if err != nil {
		log.Println(err)
		return "", err
	}
	var res NominatimResults
	if err = json.Unmarshal(data, &res); err != nil {
		log.Println(err)
		return "", err
	}
	if len(res) == 0 {
		return "", errNoPlace
	}
	p := gotz.Point{
		Lat: res[0].Lat,
		Lon: res[0].Lon,
	}
	zone, err := gotz.GetZone(p)
	if err != nil {
		return "", errNoZone
	}
	offset := time.Second * time.Duration(getOffset(target, zone))
	adress := res[0].DisplayName

	if timeNow().UTC().Add(offset).Before(target) {
		humandur := durafmt.Parse(target.Sub(timeNow().UTC().Add(offset)))
		return fmt.Sprintf(stNewYearWillHappen, adress, removeMilliseconds(humandur)), nil
	}
	humandur := durafmt.Parse(timeNow().UTC().Add(offset).Sub(target))
	return fmt.Sprintf(stNewYearHappenned, adress, removeMilliseconds(humandur)), nil
}
