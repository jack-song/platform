// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api

import (
	"fmt"

	"github.com/mattermost/platform/model"
	"github.com/mattermost/platform/utils"
)

var matterbotUser *model.User

const (
	MATTERBOT_NAME = "matterbot1"
)

func InitMatterbot() {
	// Find an existing matterbot or create a new matterbot user
	if matterbotUser == nil {
		matterbotUser = makeMatterbotUserIfNeeded()
	}
}

func makeMatterbotUserIfNeeded() *model.User {
	// Try to find an existing matterbot user
	if result := <-Srv.Store.User().GetByUsername(MATTERBOT_NAME); result.Err == nil {
		return result.Data.(*model.User)
	} else {
		// Create a new matterbot user
		newUser := &model.User{
			Email:    "matterbot@example.com",
			Username: MATTERBOT_NAME,
			Nickname: MATTERBOT_NAME,
			Password: "Password1",
		}

		if u, err := CreateUser(newUser); err != nil {
			// TODO: Handle this error
			return nil
		} else {
			return u
		}
	}
}

func SendMatterbotMessage(c *Context, userId string, message string) {
	if matterbotUser == nil {
		return
	}

	if userId == matterbotUser.Id {
		return
	}

	// Get or create direct message channel to the user from matterbot
	if sc, err := CreateDirectChannel(matterbotUser.Id, userId); err != nil {
		// TODO: Handle this error
		return
	} else {
		post := &model.Post{
			ChannelId: sc.Id,
			Message:   message,
			Type:      model.POST_DEFAULT,
			UserId:    matterbotUser.Id,
		}

		if _, err := CreatePost(c, post, false); err != nil {
			// TODO: Handle this error
		}
	}
}

func MatterbotPostUserRemovedMessage(c *Context, removedUserId string, otherUserId string, channel *model.Channel) {
	if matterbotUser == nil {
		return
	}

	// Get the user that removed the removed user
	if oresult := <-Srv.Store.User().Get(otherUserId); oresult.Err != nil {
		// TODO: Handle error
		return
	} else {
		otherUser := oresult.Data.(*model.User)
		message := fmt.Sprintf(utils.T("api.channel.remove_member.matterbot_remove"), channel.DisplayName, otherUser.Username)

		go SendMatterbotMessage(c, removedUserId, message)
	}
}
