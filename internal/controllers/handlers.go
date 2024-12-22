package controllers

import (
	"net/http"

	"github.com/bicosteve/booking-system/internal/repo"
	"github.com/bicosteve/booking-system/pkg/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
)

func (b *Base) Register(w http.ResponseWriter, r *http.Request) {
	var payload = new(entities.UserPayload)

	err := utils.SerializeJSON(w, r, payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.ValidateUser(payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = repo.CreateUser(*payload, b.Cache)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.SendMessageToKafka(b.Broker, b.Topic, b.Key, payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.DeserializeJSON(w, http.StatusOK, map[string]string{"msg": "success"})
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

}

func (b *Base) Login(w http.ResponseWriter, r *http.Request) {
	var payload = new(entities.UserPayload)

	err := utils.SerializeJSON(w, r, payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.ValidateLogin(payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.DeserializeJSON(w, http.StatusOK, payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		entities.MessageLogs.ErrorLog.Println(err)
		return
	}

}
