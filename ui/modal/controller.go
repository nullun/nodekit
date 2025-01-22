package modal

import (
	"bytes"
	"fmt"
	"github.com/algorandfoundation/nodekit/internal/algod"
	"github.com/algorandfoundation/nodekit/internal/algod/participation"
	"github.com/algorandfoundation/nodekit/ui/app"
	"github.com/algorandfoundation/nodekit/ui/modals/generate"
	"github.com/algorandfoundation/nodekit/ui/style"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"time"
)

// Init initializes the current ViewModel by batching initialization commands for all associated modal ViewModels.
func (m ViewModel) Init() tea.Cmd {
	return tea.Batch(
		m.infoModal.Init(),
		m.exceptionModal.Init(),
		m.transactionModal.Init(),
		m.confirmModal.Init(),
		m.generateModal.Init(),
	)
}

func boolToInt(input bool) int {
	if input {
		return 1
	}
	return 0
}

// HandleMessage processes the given message, updates the ViewModel state, and returns any commands to execute.
func (m ViewModel) HandleMessage(msg tea.Msg) (*ViewModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {
	case error:
		m.Open = true
		m.exceptionModal.Message = msg.Error()
		m.SetType(app.ExceptionModal)
	case participation.ShortLinkResponse:
		m.Open = true
		m.SetShortLink(msg)
		m.SetType(app.TransactionModal)
	case *algod.StateModel:
		// Clear the catchup modal
		if msg.Status.State != algod.FastCatchupState && m.Type == app.ExceptionModal && m.title == "Fast Catchup" {
			m.Open = false
			m.SetType(app.InfoModal)
		}

		m.State = msg
		m.transactionModal.State = msg
		m.infoModal.State = msg

		// On Fast-Catchup, handle the state as an exception modal
		if m.State.Status.State == algod.FastCatchupState {
			m.Open = true
			m.SetType(app.ExceptionModal)
			m.exceptionModal.Message = style.LightBlue(lipgloss.JoinVertical(lipgloss.Top,
				"Please wait while your node syncs with the network.",
				"This process can take up to an hour.",
				"",
				fmt.Sprintf("Accounts Processed:   %d / %d", m.State.Status.CatchpointAccountsProcessed, m.State.Status.CatchpointAccountsTotal),
				fmt.Sprintf("Accounts Verified:    %d / %d", m.State.Status.CatchpointAccountsVerified, m.State.Status.CatchpointAccountsTotal),
				fmt.Sprintf("Key Values Processed: %d / %d", m.State.Status.CatchpointKeyValueProcessed, m.State.Status.CatchpointKeyValueTotal),
				fmt.Sprintf("Key Values Verified:  %d / %d", m.State.Status.CatchpointKeyValueVerified, m.State.Status.CatchpointKeyValueTotal),
				fmt.Sprintf("Downloaded blocks:    %d / %d", m.State.Status.CatchpointBlocksAcquired, m.State.Status.CatchpointBlocksTotal),
				"",
				fmt.Sprintf("Sync Time: %ds", m.State.Status.SyncTime/int(time.Second)),
			))
			m.borderColor = "7"
			m.controls = ""
			m.title = "Fast Catchup"

		} else if m.Type == app.TransactionModal && m.transactionModal.Participation != nil {
			// Get the existing account from the state
			acct, ok := msg.Accounts[m.Address]
			// If the previous state is not active
			if ok {
				if !m.transactionModal.Active {
					if acct.Participation != nil && acct.Status == "Online" {
						// comparing values to detect corrupted/non-resident keys
						fvMatch := boolToInt(acct.Participation.VoteFirstValid == m.transactionModal.Participation.Key.VoteFirstValid)
						lvMatch := boolToInt(acct.Participation.VoteLastValid == m.transactionModal.Participation.Key.VoteLastValid)
						kdMatch := boolToInt(acct.Participation.VoteKeyDilution == m.transactionModal.Participation.Key.VoteKeyDilution)
						selMatch := boolToInt(bytes.Equal(acct.Participation.SelectionParticipationKey, m.transactionModal.Participation.Key.SelectionParticipationKey))
						votMatch := boolToInt(bytes.Equal(acct.Participation.VoteParticipationKey, m.transactionModal.Participation.Key.VoteParticipationKey))
						spkMatch := boolToInt(bytes.Equal(*acct.Participation.StateProofKey, *m.transactionModal.Participation.Key.StateProofKey))
						matchCount := fvMatch + lvMatch + kdMatch + selMatch + votMatch + spkMatch
						if matchCount == 6 {
							m.SetActive(true)
							m.infoModal.Active = true
							m.infoModal.Prefix = "Successfully registered online!\n"
							m.HasPrefix = true
							m.SetType(app.InfoModal)
						} else if matchCount >= 4 {
							// We use 4 as the "non resident key" threshold here
							// because it would be valid to re-reg with a key that has the same fv / lv / kd
							// but it would trigger the non resident condition
							// TOOD: refactor this beast to have {previous state} -> compare with next state
							m.SetActive(true)
							m.infoModal.Active = true
							m.infoModal.Prefix = "***WARNING***\nRegistered online but keys do not fully match\nCheck your registered keys carefully against the node keys\n\n"
							if fvMatch == 0 {
								m.infoModal.Prefix = m.infoModal.Prefix + "Mismatched: Vote First Valid\n"
							}
							if lvMatch == 0 {
								m.infoModal.Prefix = m.infoModal.Prefix + "Mismatched: Vote Last Valid\n"
							}
							if kdMatch == 0 {
								m.infoModal.Prefix = m.infoModal.Prefix + "Mismatched: Vote Key Dilution\n"
							}
							if votMatch == 0 {
								m.infoModal.Prefix = m.infoModal.Prefix + "Mismatched: Vote Key\n"
							}
							if selMatch == 0 {
								m.infoModal.Prefix = m.infoModal.Prefix + "Mismatched: Selection Key\n"
							}
							if spkMatch == 0 {
								m.infoModal.Prefix = m.infoModal.Prefix + "Mismatched: State Proof Key\n"
							}
							m.HasPrefix = true
							m.SetType(app.InfoModal)
						}
					}
				} else {
					// TODO: This includes suspended keys, where Status == offline but .Participation is set
					// Detect and display this
					if acct.Participation == nil {
						m.SetActive(false)
						m.SetType(app.InfoModal)
					} else {
						m.SetSuspended()
						m.SetType(app.InfoModal)
					}
				}
			}
		}

	case app.ModalEvent:
		if msg.Type == app.ExceptionModal {
			m.Open = true
			m.exceptionModal.Message = msg.Err.Error()
			m.generateModal.SetStep(generate.AddressStep)
			m.SetType(app.ExceptionModal)
		}

		if msg.Type == app.InfoModal {
			m.infoModal.Prefix = msg.Prefix
			m.generateModal.SetStep(generate.AddressStep)
		}
		// On closing events
		if msg.Type == app.CloseModal {
			m.Open = false
			m.generateModal.Input.Focus()
		} else {
			m.Open = true
		}
		// When something has triggered a cancel
		if msg.Type == app.CancelModal {
			switch m.Type {
			case app.InfoModal:
				m.Open = false
			case app.GenerateModal:
				m.Open = false
				m.SetType(app.InfoModal)
				m.generateModal.SetStep(generate.AddressStep)
				m.generateModal.Input.Focus()
			case app.TransactionModal:
				m.SetType(app.InfoModal)
			case app.ExceptionModal:
				m.Open = false
			case app.ConfirmModal:
				m.SetType(app.InfoModal)
			}
		}

		if msg.Type != app.CloseModal && msg.Type != app.CancelModal {
			m.SetKey(msg.Key)
			m.SetAddress(msg.Address)
			m.SetActive(msg.Active)
			m.SetType(msg.Type)
		}

	// Handle Modal Type
	case app.ModalType:
		m.SetType(msg)

	// Handle Confirmation Dialog Delete Finished
	case app.DeleteFinished:
		m.Open = false
		m.Type = app.InfoModal
		if msg.Err != nil {
			m.Open = true
			m.Type = app.ExceptionModal
			m.exceptionModal.Message = "Delete failed"
		}
	// Handle View Size changes
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		b := style.Border.Render("")
		// Custom size message
		modalMsg := tea.WindowSizeMsg{
			Width:  m.Width - lipgloss.Width(b),
			Height: m.Height - lipgloss.Height(b),
		}

		// Handle the page resize event
		m.infoModal, cmd = m.infoModal.HandleMessage(modalMsg)
		cmds = append(cmds, cmd)
		m.transactionModal, cmd = m.transactionModal.HandleMessage(modalMsg)
		cmds = append(cmds, cmd)
		m.confirmModal, cmd = m.confirmModal.HandleMessage(modalMsg)
		cmds = append(cmds, cmd)
		m.generateModal, cmd = m.generateModal.HandleMessage(modalMsg)
		cmds = append(cmds, cmd)
		m.exceptionModal, cmd = m.exceptionModal.HandleMessage(modalMsg)
		cmds = append(cmds, cmd)
		return &m, tea.Batch(cmds...)
	}

	// Only trigger modal commands when they are active
	switch m.Type {
	case app.ExceptionModal:
		m.exceptionModal, cmd = m.exceptionModal.HandleMessage(msg)
	case app.InfoModal:
		m.infoModal, cmd = m.infoModal.HandleMessage(msg)
	case app.TransactionModal:
		m.transactionModal, cmd = m.transactionModal.HandleMessage(msg)

	case app.ConfirmModal:
		m.confirmModal, cmd = m.confirmModal.HandleMessage(msg)
	case app.GenerateModal:
		m.generateModal, cmd = m.generateModal.HandleMessage(msg)
	}
	cmds = append(cmds, cmd)

	return &m, tea.Batch(cmds...)
}

// Update processes the given message, updates the ViewModel state, and returns the updated model and accompanying commands.
func (m ViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.HandleMessage(msg)
}
