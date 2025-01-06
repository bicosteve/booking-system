package controllers

import (
	"net/http"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/bicosteve/booking-system/service"
)

func (b *Base) RegisterHandler(s *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		err = s.SubmitRegistrationRequest(*payload)
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
