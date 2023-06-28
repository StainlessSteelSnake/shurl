package storage

const (
	queryInsert = `
	INSERT INTO public.short_urls
	    (
			short_url, long_url, user_id
		)
	VALUES ($1, $2, $3);`

	queryCreateTable = `
	CREATE TABLE IF NOT EXISTS public.short_urls
		(
			short_url character varying(14) COLLATE pg_catalog."default" NOT NULL,
			long_url character varying COLLATE pg_catalog."default" NOT NULL,
			user_id character varying COLLATE pg_catalog."default",
			deleted boolean NOT NULL DEFAULT false,
			
			CONSTRAINT short_urls_pkey PRIMARY KEY (short_url)
		)	
	TABLESPACE pg_default;

	CREATE UNIQUE INDEX IF NOT EXISTS unique_long_url
    	ON public.short_urls USING btree
    (	long_url COLLATE pg_catalog."default" ASC NULLS LAST, 
    	deleted  ASC NULLS LAST	) 
    TABLESPACE pg_default;
`

	querySelectAll = `
	SELECT short_url, long_url, user_id, deleted
	FROM short_urls`

	querySelectByLongURL = `SELECT short_url FROM short_urls WHERE long_url = $1 AND deleted <> true`

	queryDelete = `UPDATE short_urls SET deleted = true WHERE short_url = $1`
)
