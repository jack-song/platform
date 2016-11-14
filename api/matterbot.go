// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package api

import (
	"fmt"

	l4g "github.com/alecthomas/log4go"
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

	// Try to get an existing direct channel
	var botchannel *model.Channel
	if result := <-Srv.Store.Channel().GetByName("", model.GetDMNameFromIds(userId, matterbotUser.Id)); result.Err != nil {
		// Create a direct channel
		if sc, err := CreateDirectChannel(matterbotUser.Id, userId); err != nil {
			// TODO: Handle this error
		} else {
			botchannel = sc
		}
	} else {
		botchannel = result.Data.(*model.Channel)
	}

	// Create the post
	if botchannel != nil {
		post := &model.Post{
			ChannelId: botchannel.Id,
			Message:   message,
			Type:      model.POST_DEFAULT,
			UserId:    matterbotUser.Id,
		}

		if _, err := CreatePost(c, post, false); err != nil {
			// TODO: Handle this error
		} else {
			// Ensure that the matterbot channel is being shown
			go showMatterbotDirectChannel(userId)
		}
	}
}

func showMatterbotDirectChannel(userId string) {
	if matterbotUser == nil {
		return
	}

	var preference *model.Preference
	if result := <-Srv.Store.Preference().Get(userId, model.PREFERENCE_CATEGORY_DIRECT_CHANNEL_SHOW, matterbotUser.Id); result.Err != nil {
		// Create a new preference to show the matterbot channel
		preference = &model.Preference{
			UserId:   userId,
			Category: model.PREFERENCE_CATEGORY_DIRECT_CHANNEL_SHOW,
			Name:     matterbotUser.Id,
			Value:    "true",
		}
	} else if existingPref := result.Data.(model.Preference); existingPref.Value != "true" {
		// Change the preference to show the direct channel
		preference = &existingPref
		preference.Value = "true"
	}

	// Save the updated preference if we need to
	if preference != nil {
		if saveResult := <-Srv.Store.Preference().Save(&model.Preferences{*preference}); saveResult.Err != nil {
			// TODO: Handle error
		} else {
			// Notify that the user's preferences have been changed
			message := model.NewWebSocketEvent(model.WEBSOCKET_EVENT_PREFERENCE_CHANGED, "", "", userId, nil)
			message.Add("preference", preference.ToJson())

			go Publish(message)
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
		message := fmt.Sprintf(utils.T("api.matterbot.channel.remove_member.removed"), channel.DisplayName, otherUser.Username)

		go SendMatterbotMessage(c, removedUserId, message)
	}
}

func MatterbotPostChannelDeletedMessage(c *Context, channel *model.Channel, user *model.User) {
	var members []model.ChannelMember

	if result := <-Srv.Store.Channel().GetMembers(channel.Id); result.Err != nil {
		l4g.Error(utils.T("api.matterbot.channel.retrieve_members.error"), channel.Id)
		return
	} else {
		members = result.Data.([]model.ChannelMember)

		for _, channelMember := range members {
			if channelMember.UserId != user.Id {
				go SendMatterbotMessage(c, channelMember.UserId, fmt.Sprintf(utils.T("api.matterbot.channel.delete_channel.archived"), user.Username, channel.DisplayName))
			}
		}
	}
}
