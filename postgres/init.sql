-- 
-- Name: message; Type: TABLE; Schema: public; Owner: postgres
--
CREATE TABLE public.message (
        -- message id
        id bigserial PRIMARY KEY,
        -- device id
        device_id varchar(36),
        -- message topic
        topic text NOT NULL,
        -- message payload
        payload jsonb NOT NULL,
        -- create time
        create_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 
-- Name: connection; Type: TABLE; Schema: public; Owner: postgres
--
CREATE TABLE public.connection (
        -- device id
        id varchar(36) PRIMARY KEY,
        -- is stopped
        stopped boolean DEFAULT FALSE,
        -- connection config
        payload jsonb NOT NULL,
        -- create time
        create_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
        -- update time
        update_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);