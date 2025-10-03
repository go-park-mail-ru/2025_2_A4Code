```mermaid
erDiagram
    PROFILE {
        Id PK
        Login "AK"
        PasswordHash
        Name
        Surname
        Patronymic
        Gender
        Birthday
        AvatarId FK
        PhoneNumber "AK"
        AuthVersion
    }
    MESSAGE {
        Id PK
        Topic
        Text
        DateOfDispatch
        SenderId FK
        ThreadId FK
        IsRead
    }
    THREAD {
        Id PK
        RootMessageId FK
    }
    RECIPIENT {
        Id PK
        MessageId FK
        Address
    }
    FILE {
        Id PK
        FileType
        Size
        StoragePath
    }
    MESSAGEFILE {
        Id PK
        MessageId FK
        FileId FK
    }
    PROFILEMESSAGE {
        ProfileId PK "FK"
        MessageId PK "FK"
        ReadStatus
        DeletedStatus
        DraftStatus
    }
    FOLDER {
        Id PK
        ProfileId FK "AK"
        Name "AK"
        Type  " 'custom', 'inbox', 'sent', 'trash', etc. "
    }
    FOLDERMESSAGE {
        FolderId PK "FK"
        MessageId PK "FK"
    }
    SETTINGS {
        Id PK
        ProfileId FK "AK"
        NotificationTolerance
        Language
        Theme
        Signature
    }

    PROFILE ||--o| SETTINGS : "has"
    PROFILE ||--o{ FOLDER : "owns"
    FOLDER ||--o{ FOLDERMESSAGE : "contains"
    MESSAGE ||--o{ FOLDERMESSAGE : "locatedIn"
    MESSAGE ||--o{ PROFILEMESSAGE : "relatedTo"
    PROFILE ||--o{ PROFILEMESSAGE : "receivedBy"
    MESSAGE ||--o{ MESSAGEFILE : "attaches"
    FILE ||--o{ MESSAGEFILE : "attachedTo"
    PROFILE ||--o| FILE : "hasAvatar"
    MESSAGE ||--o{ RECIPIENT : "hasRecipient"
    THREAD ||--o{ MESSAGE : "groups"
```