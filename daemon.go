package main

import (
	"encoding/json"
	"fmt"
	"github.com/leoleovich/3djuggler/juggler"
	"github.com/leoleovich/go-gcodefeeder/gcodefeeder"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type Daemon struct {
	timer      *time.Timer
	config     *Config
	jobfile         string
	job             *juggler.Job
	ie              *InternEnpoint
	feeder          *gcodefeeder.Feeder
}

func (daemon *Daemon) UpdateStatus(status juggler.JobStatus) {
	daemon.job.Status = status

	// Don't send to intern this status
	if status == juggler.StatusWaitingJob {
		return
	}

	log.Infof("Updating intern status to %s", status)
	if err := daemon.ie.reportJobStatusChange(daemon.job); err != nil {
		log.Error("Can't report it to intern: ", err)
	}
}

// InfoHandler gives provides with json containing job status and some other important fields
func (daemon *Daemon) InfoHandler(w http.ResponseWriter, r *http.Request) {
	// Add headers to allow AJAX
	juggler.SetHeaders(w)

	b, err := json.Marshal(daemon.job)
	if err != nil {
		log.Errorf("Failed to respond on /info request: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, string(b))
}

// StartHandler acknowledged start of the job
func (daemon *Daemon) StartHandler(w http.ResponseWriter, r *http.Request) {
	// Add headers to allow AJAX
	juggler.SetHeaders(w)

	if daemon.job.Status != juggler.StatusWaitingButton {
		errS := fmt.Sprintf("Ignore buttonpress in '%v' status", daemon.job.Status)
		log.Infof(errS)
		http.Error(w, errS, http.StatusBadRequest)
		return
	}

	daemon.UpdateStatus(juggler.StatusSending)
}

// RescheduleHandler resets the time when the job will start
func (daemon *Daemon) RescheduleHandler(w http.ResponseWriter, r *http.Request) {
	// Add headers to allow AJAX
	juggler.SetHeaders(w)

	if daemon.job.Status != juggler.StatusWaitingButton {
		errS := fmt.Sprintf("Ignore reschedule in '%v' status", daemon.job.Status)
		log.Infof(errS)
		http.Error(w, errS, http.StatusBadRequest)
		return
	}

	daemon.job.Fetched = time.Now()
	daemon.job.Scheduled = time.Now().Add(waitingForButtonInterval)
}

// CancelHandler cancels job execution
func (daemon *Daemon) CancelHandler(w http.ResponseWriter, r *http.Request) {
	// Add headers to allow AJAX
	juggler.SetHeaders(w)

	if daemon.job.Id == 0 {
		errS := fmt.Sprintf("Ignore cancel, no job scheduled")
		log.Infof(errS)
		http.Error(w, errS, http.StatusBadRequest)
		return
	}

	daemon.job.Scheduled = time.Time{}
	daemon.UpdateStatus(juggler.StatusCancelling)
}