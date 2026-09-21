import { gql } from "@apollo/client";

export const ME = gql`
  query Me {
    me {
      id
      email
      name
      role
    }
    oidcLabel
  }
`;

export const HOME = gql`
  query Home($city: String, $category: String, $window: TimeWindow) {
    home(city: $city, category: $category, window: $window) {
      city
      rankedStores {
        rank
        communityScore
        feedScore
        store {
          id
          slug
          name
          description
          category
          city
          upvoteCount
          viewerHasVoted
          createdAt
          photos {
            id
            url
          }
        }
      }
      trendingProducts {
        heat
        product {
          id
          slug
          name
          displayPrice
          photos {
            url
          }
          store {
            slug
            name
            category
            city
          }
        }
      }
      trendingReviews {
        heat
        review {
          id
          rating
          body
          authorFirstName
          createdAt
          store {
            slug
            name
            city
          }
          product {
            slug
            name
          }
        }
      }
    }
  }
`;

export const SEARCH = gql`
  query Search($q: String!, $city: String) {
    search(q: $q, city: $city) {
      stores {
        id
        slug
        name
        description
        category
        city
        photos {
          url
        }
      }
      products {
        id
        slug
        name
        displayPrice
        description
        photos {
          url
        }
        store {
          slug
          name
        }
      }
    }
  }
`;

export const STORE = gql`
  query Store($slug: String!) {
    store(slug: $slug) {
      id
      slug
      name
      description
      category
      city
      phone
      address
      upvoteCount
      viewerHasVoted
      createdAt
      photos {
        id
        url
      }
      products {
        id
        slug
        name
        description
        displayPrice
        photos {
          url
        }
      }
      reviews {
        id
        rating
        body
        authorFirstName
        createdAt
      }
    }
  }
`;

export const PRODUCT = gql`
  query Product($slug: String!) {
    product(slug: $slug) {
      id
      slug
      name
      description
      displayPrice
      priceCents
      upvoteCount
      viewerHasVoted
      photos {
        url
      }
      store {
        id
        slug
        name
        category
        city
        phone
      }
      reviews {
        id
        rating
        body
        authorFirstName
        createdAt
      }
    }
  }
`;

export const MY_STORES = gql`
  query MyStores {
    myStores {
      id
      slug
      name
      description
      category
      city
      phone
      address
      photos {
        url
      }
      products {
        id
        slug
        name
        description
        priceCents
        displayPrice
      }
    }
  }
`;

export const SIGNUP = gql`
  mutation Signup($email: String!, $password: String!, $name: String!, $role: Role) {
    signup(email: $email, password: $password, name: $name, role: $role) {
      id
      email
      name
      role
    }
  }
`;

export const LOGIN = gql`
  mutation Login($email: String!, $password: String!) {
    login(email: $email, password: $password) {
      id
      email
      name
      role
    }
  }
`;

export const LOGOUT = gql`
  mutation Logout {
    logout
  }
`;

export const BECOME_OWNER = gql`
  mutation BecomeOwner {
    becomeOwner {
      id
      role
      name
    }
  }
`;

export const VOTE = gql`
  mutation Vote($storeId: ID, $productId: ID) {
    vote(storeId: $storeId, productId: $productId)
  }
`;

export const CREATE_REVIEW = gql`
  mutation CreateReview($storeId: ID, $productId: ID, $rating: Int!, $body: String!) {
    createReview(storeId: $storeId, productId: $productId, rating: $rating, body: $body) {
      id
      rating
      body
      authorFirstName
      createdAt
    }
  }
`;

export const RECORD_AFFINITY = gql`
  mutation RecordAffinity($category: String!, $kind: AffinityEventKind!) {
    recordAffinityEvent(category: $category, kind: $kind)
  }
`;

export const CHAT = gql`
  mutation Chat($storeId: ID, $productId: ID, $threadId: ID, $message: String!) {
    chat(storeId: $storeId, productId: $productId, threadId: $threadId, message: $message) {
      threadId
      message
      phone
    }
  }
`;

export const CREATE_STORE = gql`
  mutation CreateStore($input: CreateStoreInput!) {
    createStore(input: $input) {
      id
      slug
      name
    }
  }
`;

export const UPDATE_STORE = gql`
  mutation UpdateStore($id: ID!, $input: UpdateStoreInput!) {
    updateStore(id: $id, input: $input) {
      id
      slug
      name
    }
  }
`;

export const DELETE_STORE = gql`
  mutation DeleteStore($id: ID!) {
    deleteStore(id: $id)
  }
`;

export const CREATE_PRODUCT = gql`
  mutation CreateProduct($storeId: ID!, $input: CreateProductInput!) {
    createProduct(storeId: $storeId, input: $input) {
      id
      slug
      name
    }
  }
`;

export const UPDATE_PRODUCT = gql`
  mutation UpdateProduct($id: ID!, $input: UpdateProductInput!) {
    updateProduct(id: $id, input: $input) {
      id
      slug
    }
  }
`;

export const UPLOAD_PHOTO = gql`
  mutation UploadPhoto($storeId: ID, $productId: ID, $filename: String!, $contentBase64: String!) {
    uploadPhoto(storeId: $storeId, productId: $productId, filename: $filename, contentBase64: $contentBase64) {
      id
      url
    }
  }
`;
