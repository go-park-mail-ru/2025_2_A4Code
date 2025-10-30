-- Вставка начальных данных
DO $$
DECLARE
bp1 INTEGER;
    bp2 INTEGER;
    bp3 INTEGER;
    bp4 INTEGER;
    p1 INTEGER;
    p2 INTEGER;
    p3 INTEGER;
    p4 INTEGER;
    t1 INTEGER;
    t2 INTEGER;
    t3 INTEGER;
    t4 INTEGER;
    f1 INTEGER;
    f2 INTEGER;
    f3 INTEGER;
    f4 INTEGER;
    f5 INTEGER;
    f6 INTEGER;
    m1 INTEGER;
    m2 INTEGER;
    m3 INTEGER;
    m4 INTEGER;
BEGIN
    -- Вставка в base_profile
INSERT INTO base_profile (username, domain)
VALUES ('alexey', 'a4mail.ru') RETURNING id INTO bp1;

INSERT INTO base_profile (username, domain)
VALUES ('antonina', 'a4mail.ru') RETURNING id INTO bp2;

INSERT INTO base_profile (username, domain)
VALUES ('andrey', 'a4mail.ru') RETURNING id INTO bp3;

INSERT INTO base_profile (username, domain)
VALUES ('anna', 'a4mail.ru') RETURNING id INTO bp4;

-- Вставка в profile
INSERT INTO profile (base_profile_id, password_hash, name, surname, patronymic, gender, birthday, phone_number)
VALUES (bp1, '$2a$10$4PcooWbEMRjvdk2cMFumO.ajWaAclawIljtlfu2.2f5/fV8LkgEZe', 'Alexey', 'Gusev', 'Nikolaevich', 'Male', '2003-08-20', '+77777777777') RETURNING id INTO p1;

INSERT INTO profile (base_profile_id, password_hash,  name, surname, patronymic, gender, birthday, phone_number)
VALUES (bp2, '$2a$10$4PcooWbEMRjvdk2cMFumO.ajWaAclawIljtlfu2.2f5/fV8LkgEZe', 'Antonina', 'Andreeva', 'Aleksandrovna', 'Female', '2003-10-17', '+79697045539') RETURNING id INTO p2;

INSERT INTO profile (base_profile_id, password_hash, name, surname, patronymic, gender, birthday, phone_number)
VALUES (bp3, '$2a$10$4PcooWbEMRjvdk2cMFumO.ajWaAclawIljtlfu2.2f5/fV8LkgEZe', 'Andrey', 'Vavilov', 'Nikolaevich', 'Male', '2003-08-20', '+79099099090') RETURNING id INTO p3;

INSERT INTO profile (base_profile_id, password_hash, name, surname, patronymic, gender, birthday, phone_number)
VALUES (bp4, '$2a$10$4PcooWbEMRjvdk2cMFumO.ajWaAclawIljtlfu2.2f5/fV8LkgEZe', 'Anna', 'Mihonina', 'Aleksandrovna', 'Female', '2003-08-20', '+79099499090') RETURNING id INTO p4;

-- Вставка в thread
INSERT INTO thread (root_message_id) VALUES (NULL) RETURNING id INTO t1;
INSERT INTO thread (root_message_id) VALUES (NULL) RETURNING id INTO t2;
INSERT INTO thread (root_message_id) VALUES (NULL) RETURNING id INTO t3;
INSERT INTO thread (root_message_id) VALUES (NULL) RETURNING id INTO t4;

-- Вставка в message
INSERT INTO message (topic, text, sender_base_profile_id, thread_id)
VALUES ('Topic1 Lorem ipsum.', 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.', bp1, t1) RETURNING id INTO m1;

INSERT INTO message (topic, text, sender_base_profile_id, thread_id)
VALUES ('Topic2 Lorem ipsum.', 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.', bp1, t2) RETURNING id INTO m2;

INSERT INTO message (topic, text, sender_base_profile_id, thread_id)
VALUES ('Topic3 Lorem ipsum.', 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.', bp2, t3) RETURNING id INTO m3;

INSERT INTO message (topic, text, sender_base_profile_id, thread_id)
VALUES ('Topic4 Lorem ipsum.', 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.', bp2, t4) RETURNING id INTO m4;

-- Обновление root_message_id в thread
UPDATE thread SET root_message_id = m1 WHERE id = t1;
UPDATE thread SET root_message_id = m2 WHERE id = t2;
UPDATE thread SET root_message_id = m3 WHERE id = t3;
UPDATE thread SET root_message_id = m4 WHERE id = t4;

-- Вставка в folder
INSERT INTO folder (profile_id, folder_name, folder_type)
VALUES (p1, 'Inbox', 'inbox') RETURNING id INTO f1;

INSERT INTO folder (profile_id, folder_name, folder_type)
VALUES (p1, 'Sent', 'sent') RETURNING id INTO f2;

INSERT INTO folder (profile_id, folder_name, folder_type)
VALUES (p2, 'Inbox', 'inbox') RETURNING id INTO f3;

INSERT INTO folder (profile_id, folder_name, folder_type)
VALUES (p2, 'Sent', 'sent') RETURNING id INTO f4;

INSERT INTO folder (profile_id, folder_name, folder_type)
VALUES (p3, 'Inbox', 'inbox') RETURNING id INTO f5;

INSERT INTO folder (profile_id, folder_name, folder_type)
VALUES (p4, 'Inbox', 'inbox') RETURNING id INTO f6;

-- Вставка в folder_profile_message
INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m1, f2);
INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m1, f3);
INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m2, f2);
INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m2, f5);
INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m3, f4);
INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m3, f5);
INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m3, f6);
INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m4, f4);
INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m4, f6);

-- Вставка в profile_message
INSERT INTO profile_message (profile_id, message_id) VALUES (p1, m1);
INSERT INTO profile_message (profile_id, message_id) VALUES (p2, m1);
INSERT INTO profile_message (profile_id, message_id) VALUES (p1, m2);
INSERT INTO profile_message (profile_id, message_id) VALUES (p3, m2);
INSERT INTO profile_message (profile_id, message_id) VALUES (p2, m3);
INSERT INTO profile_message (profile_id, message_id) VALUES (p3, m3);
INSERT INTO profile_message (profile_id, message_id) VALUES (p4, m3);
INSERT INTO profile_message (profile_id, message_id) VALUES (p2, m4);
INSERT INTO profile_message (profile_id, message_id) VALUES (p4, m4);

-- Вставка в settings
INSERT INTO settings (profile_id) VALUES (p1);
INSERT INTO settings (profile_id) VALUES (p2);
INSERT INTO settings (profile_id) VALUES (p3);
INSERT INTO settings (profile_id) VALUES (p4);
END $$;