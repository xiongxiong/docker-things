CREATE TABLE public.message (
        id bigserial PRIMARY KEY,
        -- subscription id (uuid)
        subscription_id char(36),
        -- message topic
        topic text NOT NULL,
        -- message payload
        payload jsonb NOT NULL
);
CREATE INDEX idx_subid_topic ON messages (subscription_id, topic);
CREATE INDEX idxbody ON messages USING GIN (payload);

CREATE TABLE public.subscription (
        -- id (uuid)
        id char(36) PRIMARY KEY,
        -- is subscription open
        isOpen boolean,
        username text,
        password text,
        -- broker list
        brokers text,
        -- topic list
        topicPairs text
);