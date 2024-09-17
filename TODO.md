# TODO

- feedback when assistant is being updated
- list facts by box
- delete "box"
- (?) some equivalent of `cmd`


# Bugs

## Logical

## UI/UX
- cursor highlight retained on screen when in selection mode

## Textsel
- feature: `/` to search
- bug: page up/down is not moving the cursor onto the visible screen
- bug: format codes showing at cursor location when cursor is at the beginning of a change in formatting
- bug: when copying text that contains square brackets, they are being stripped from the paste buffer output because tview is interpreting them as format codes
