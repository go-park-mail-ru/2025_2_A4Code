```mermaid
erDiagram
    BASEPRO
    PROFILE {
        Id PK
        Login "AK"
        PasswordHash
        Name
        Surname
        Patronymic
        Gender
        Birthday
        ImagePath FK
        PhoneNumber "AK"
        AuthVersion
        CreatedAt
        UpdatedAt
    }
        FOREINPROFILE {
        Id PK
        Login "AK"
        Name
        Surname
        Patronymic
        Gender
        Birthday
        AvatarId FK
        PhoneNumber "AK"
        AuthVersion 
        CreatedAt
        UpdatedAt
    }
    MESSAGE {
        Id PK
        Topic
        Text
        DateOfDispatch
        SenderId FK
        ThreadId FK
        IsRead
        CreatedAt
        UpdatedAt
    }
    THREAD {
        Id PK
        RootMessageId FK
        CreatedAt
        UpdatedAt
    }
    FILE {
        Id PK
        FileType
        Size
        StoragePath
        MessageId
        CreatedAt
        UpdatedAt
    }
    PROFILEMESSAGE {
        ProfileId PK "FK"
        MessageId PK "FK"
        ReadStatus
        DeletedStatus
        DraftStatus
        CreatedAt
        UpdatedAt
    }
    FOLDER {
        Id PK
        ProfileId FK "AK"
        Name "AK"
        Type  " 'custom', 'inbox', 'sent', 'trash', etc. "
        CreatedAt
        UpdatedAt
    }
    FOLDERMESSAGE {
        FolderId PK "FK"
        MessageId PK "FK"
        CreatedAt
        UpdatedAt
    }
    SETTINGS {
        Id PK
        ProfileId FK "AK"
        NotificationTolerance
        Language
        Theme
        Signature
        CreatedAt
        UpdatedAt
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