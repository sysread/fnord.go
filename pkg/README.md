# Conversation Model

There are multiple components that tie together to manage conversations.

## messages

The messages package contains the basic structure of a message and a
conversation.

## data

This is the persistence layer. It wraps the `messages` module `Conversation`
struct, adding metadata about the time the conversation was created and last
updated, a space to store a summary of the conversation, as well as management
of the embedding (of the summary) generated to make the conversation
searchable.

It also provides the logic for saving and searching old conversation data.

## chat_manager

This is the main entry point for the conversation model. It provides the
top-level control flow for the conversation as a whole, including the prompts
used to request assistant responses, summarize the conversation, and generate
the search queries used to find information from earlier conversations.

## gpt

This module provides the interface to the OpenAI API. It is intended to only
manage the actual API calls and responses, without any business logic tied to
the conversation itself.
