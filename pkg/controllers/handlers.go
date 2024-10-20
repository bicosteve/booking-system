package controllers

import (
	"net/http"

	"github.com/bicosteve/booking-system/pkg/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
)

func (b *Base) Register(w http.ResponseWriter, r *http.Request) {
	var payload = new(entities.UserPayload)

	err := utils.SerializeJSON(w, r, payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.ValidateUser(payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.DeserializeJSON(w, http.StatusOK, payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.MessageLogs.ErrorLog.Println(err)
		return
	}

}

func (b *Base) Login(w http.ResponseWriter, r *http.Request) {
	var payload = new(entities.UserPayload)

	err := utils.SerializeJSON(w, r, payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.ValidateLogin(payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.MessageLogs.ErrorLog.Println(err)
		return
	}

	err = utils.DeserializeJSON(w, http.StatusOK, payload)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		utils.MessageLogs.ErrorLog.Println(err)
		return
	}

}
