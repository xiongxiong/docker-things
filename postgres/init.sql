CREATE TABLE public.message (
        id bigserial PRIMARY KEY,
        -- client id (uuid)
        clientID char(36),
        -- message topic
        topic text NOT NULL,
        -- message payload
        payload jsonb NOT NULL,
        -- timestamp
        createdAt timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_client_topic ON public.message (clientID, topic);
CREATE INDEX idx_payload ON public.message USING GIN (payload);

CREATE TABLE public.client (
        -- id (uuid)
        id char(36) PRIMARY KEY,
        -- is stopped
        stopped boolean DEFAULT FALSE,
        -- system user id
        userID text NOT NULL,
        -- client config
        payload jsonb NOT NULL,
        -- timestamp
        createdAt timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_payload ON public.client USING GIN (payload);