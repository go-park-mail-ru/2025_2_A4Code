-- revert removal of unique(username) if needed
ALTER TABLE base_profile
    ADD CONSTRAINT base_profile_username_key UNIQUE (username);
